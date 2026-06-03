package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// Edge-TTS: memanfaatkan layanan "read aloud" neural milik Microsoft Edge yang
// gratis & tanpa API key. Default suara wanita Bahasa Indonesia (Gadis Neural).
const (
	edgeTrustedToken  = "6A5AA1D4EAFF4E9FB37E23D68491D6F4"
	edgeWSSURL        = "wss://speech.platform.bing.com/consumer/speech/synthesize/readaloud/edge/v1"
	edgeSecGECVersion = "1-140.0.3485.14" // versi <133 ditolak Microsoft per 2026
	edgeOutputFormat  = "audio-24khz-48kbitrate-mono-mp3"
	edgeUserAgent     = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0.0.0 Safari/537.36 Edg/140.0.0.0"
	edgeOrigin        = "chrome-extension://jdiccldimpdaibmpdkjnbmckianbfold"

	defaultEdgeVoice    = "id-ID-GadisNeural" // suara wanita Bahasa Indonesia
	defaultEdgeLanguage = "id-ID"
	edgeRate            = "-5%" // tempo sedikit tenang seperti pengumuman bandara

	winEpochSeconds = 11644473600 // detik antara 1601-01-01 dan 1970-01-01
	ttsCacheMax     = 256
)

// TTSConfig holds the voice selection. Edge-TTS tidak butuh API key.
type TTSConfig struct {
	Voice    string
	Language string
}

// TTSService synthesizes speech via Microsoft Edge's free neural TTS over a
// WebSocket. Hasil di-cache di memori berdasarkan teks agar panggilan ulang
// (recall) tidak menembak layanan lagi.
type TTSService struct {
	cfg TTSConfig

	mu    sync.Mutex
	cache map[string][]byte
	order []string
}

func NewTTSService(cfg TTSConfig) *TTSService {
	if cfg.Voice == "" {
		cfg.Voice = defaultEdgeVoice
	}
	if cfg.Language == "" {
		cfg.Language = defaultEdgeLanguage
	}
	return &TTSService{cfg: cfg, cache: make(map[string][]byte)}
}

// Enabled selalu true — edge-tts gratis tanpa konfigurasi. Disediakan agar
// handler tetap punya jalur fallback bila suatu saat dinonaktifkan.
func (s *TTSService) Enabled() bool { return true }

// Synthesize returns MP3 audio (audio/mpeg) for text. Mengembalikan hasil cache
// bila tersedia.
func (s *TTSService) Synthesize(ctx context.Context, text string) ([]byte, error) {
	if cached := s.getCache(text); cached != nil {
		return cached, nil
	}

	audio, err := s.synthEdge(ctx, text)
	if err != nil {
		return nil, err
	}
	if len(audio) == 0 {
		return nil, fmt.Errorf("edge-tts: audio kosong")
	}

	s.putCache(text, audio)
	return audio, nil
}

func (s *TTSService) synthEdge(ctx context.Context, text string) ([]byte, error) {
	dialCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	connURL := fmt.Sprintf(
		"%s?TrustedClientToken=%s&Sec-MS-GEC=%s&Sec-MS-GEC-Version=%s&ConnectionId=%s",
		edgeWSSURL, edgeTrustedToken, generateSecMSGEC(), edgeSecGECVersion, randomHex(16),
	)

	c, resp, err := websocket.Dial(dialCtx, connURL, &websocket.DialOptions{
		HTTPHeader: map[string][]string{
			"Pragma":          {"no-cache"},
			"Cache-Control":   {"no-cache"},
			"Origin":          {edgeOrigin},
			"Accept-Encoding": {"gzip, deflate, br"},
			"Accept-Language": {"en-US,en;q=0.9"},
			"User-Agent":      {edgeUserAgent},
		},
	})
	if err != nil {
		if resp != nil {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
			resp.Body.Close()
			return nil, fmt.Errorf("edge-tts dial: %w | resp: %s", err, strings.TrimSpace(string(body)))
		}
		return nil, fmt.Errorf("edge-tts dial: %w", err)
	}
	defer c.Close(websocket.StatusNormalClosure, "")
	c.SetReadLimit(16 << 20) // 16MB

	now := time.Now().UTC().Format("Mon Jan 02 2006 15:04:05 GMT+0000 (Coordinated Universal Time)")

	// 1) Konfigurasi sesi (format output audio).
	cfgMsg := "X-Timestamp:" + now + "\r\n" +
		"Content-Type:application/json; charset=utf-8\r\n" +
		"Path:speech.config\r\n\r\n" +
		`{"context":{"synthesis":{"audio":{"metadataoptions":{"sentenceBoundaryEnabled":"false","wordBoundaryEnabled":"false"},"outputFormat":"` +
		edgeOutputFormat + `"}}}}`
	if err := c.Write(dialCtx, websocket.MessageText, []byte(cfgMsg)); err != nil {
		return nil, fmt.Errorf("edge-tts write config: %w", err)
	}

	// 2) Kirim SSML berisi teks + pilihan suara.
	ssml := fmt.Sprintf(
		"<speak version='1.0' xmlns='http://www.w3.org/2001/10/synthesis' xml:lang='%s'>"+
			"<voice name='%s'><prosody pitch='+0Hz' rate='%s' volume='+0%%'>%s</prosody></voice></speak>",
		s.cfg.Language, s.cfg.Voice, edgeRate, escapeXML(text),
	)
	ssmlMsg := "X-RequestId:" + randomHex(16) + "\r\n" +
		"Content-Type:application/ssml+xml\r\n" +
		"X-Timestamp:" + now + "Z\r\n" +
		"Path:ssml\r\n\r\n" + ssml
	if err := c.Write(dialCtx, websocket.MessageText, []byte(ssmlMsg)); err != nil {
		return nil, fmt.Errorf("edge-tts write ssml: %w", err)
	}

	// 3) Terima frame audio sampai turn.end.
	var audio []byte
	for {
		typ, data, err := c.Read(dialCtx)
		if err != nil {
			return nil, fmt.Errorf("edge-tts read: %w", err)
		}
		switch typ {
		case websocket.MessageBinary:
			if len(data) < 2 {
				continue
			}
			headerLen := int(binary.BigEndian.Uint16(data[:2]))
			if 2+headerLen > len(data) {
				continue
			}
			if strings.Contains(string(data[2:2+headerLen]), "Path:audio") {
				audio = append(audio, data[2+headerLen:]...)
			}
		case websocket.MessageText:
			if strings.Contains(string(data), "Path:turn.end") {
				return audio, nil
			}
		}
	}
}

// generateSecMSGEC menghitung token Sec-MS-GEC seperti yang dipakai Edge:
// SHA256 dari (windows-ticks dibulatkan ke 5 menit) + trusted token.
// Perhitungan sengaja memakai float64 agar identik dengan implementasi
// referensi (nilai 18 digit melampaui presisi eksak int → harus float).
func generateSecMSGEC() string {
	ticks := float64(time.Now().Unix()) + winEpochSeconds
	ticks -= math.Mod(ticks, 300) // bulatkan ke bawah per 5 menit
	ticks *= 1e7                   // detik -> interval 100ns
	s := fmt.Sprintf("%.0f%s", ticks, edgeTrustedToken)
	sum := sha256.Sum256([]byte(s))
	return strings.ToUpper(hex.EncodeToString(sum[:]))
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func escapeXML(s string) string {
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"'", "&apos;",
		`"`, "&quot;",
	)
	return r.Replace(s)
}

func (s *TTSService) getCache(text string) []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cache[text]
}

func (s *TTSService) putCache(text string, audio []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.cache[text]; exists {
		return
	}
	// Evict entri terlama saat penuh (FIFO sederhana).
	if len(s.order) >= ttsCacheMax {
		oldest := s.order[0]
		s.order = s.order[1:]
		delete(s.cache, oldest)
	}
	s.cache[text] = audio
	s.order = append(s.order, text)
}
