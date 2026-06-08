package domain

// Counter is the wire shape consumed by the frontend (lib/constants.ts).
// JSON tags use the frontend names (active/service), not the DB column names.
type Counter struct {
	ID      int     `json:"id"`
	Name    string  `json:"name"`
	Service *string `json:"service"`
	Active  bool    `json:"active"`
	Staff   *string `json:"staff"`
	// StaffID is the assigned loket operator. Exposed so the staff panel can
	// bind a logged-in operator to their own counter (call-ownership check).
	StaffID *string `json:"staff_id"`
}
