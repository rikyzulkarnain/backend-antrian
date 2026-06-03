package handler

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/bbpjn-sumsel/sistem-antrian/internal/service"
)

const ttsMaxRunes = 300

type TTSHandler struct {
	svc *service.TTSService
}

func NewTTSHandler(svc *service.TTSService) *TTSHandler {
	return &TTSHandler{svc: svc}
}

// Speak streams MP3 audio for the given text via ElevenLabs.
// Endpoint publik (dikonsumsi layar display): GET /api/v1/tts?text=...
func (h *TTSHandler) Speak(w http.ResponseWriter, r *http.Request) {
	text := strings.TrimSpace(r.URL.Query().Get("text"))
	if text == "" {
		badRequest(w, "Parameter 'text' wajib diisi")
		return
	}
	if len([]rune(text)) > ttsMaxRunes {
		badRequest(w, "Teks terlalu panjang")
		return
	}
	if !h.svc.Enabled() {
		// Frontend akan fallback ke Web Speech bila ini terjadi.
		writeError(w, http.StatusServiceUnavailable, "TTS_DISABLED", "Layanan TTS belum dikonfigurasi")
		return
	}

	audio, err := h.svc.Synthesize(r.Context(), text)
	if err != nil {
		slog.Error("tts synthesize failed", "err", err)
		writeError(w, http.StatusBadGateway, "TTS_FAILED", "Gagal membuat audio")
		return
	}

	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Header().Set("Content-Length", strconv.Itoa(len(audio)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(audio)
}
