package ui

// SaveFormat names the output format of a SAVE request. SAV is the only
// format; any other value is rejected before preparation.
type SaveFormat string

// SaveSAV is the only save format.
const SaveSAV SaveFormat = "SAV"

// SaveRequest names one SAVE. Label, when set, is the label written into the
// file; otherwise the name is.
type SaveRequest struct {
	OnMap bool
	// Admission asks only whether this point admits a SAVE: Prepare returns
	// the refusal the player reads, or nil, and builds no file.
	Admission       bool `json:",omitempty"`
	Directory, Name string
	Label           string `json:",omitempty"`
	Format          SaveFormat
}

// Directories contains child directory names. Entries retain their disk names
// so choosing a row can fill the name field without parsing a displayed label.
type SaveDirectory struct {
	Path        string
	Directories []string
	Entries     []SaveEntry
}

// PreparedSave holds complete detached output. Existing names the exact
// paths requiring confirmation. Commit refuses a changed target or directory
// and publishes the whole selected set; cancellation simply discards this
// value. Notice, when set, names a disclosed SAV approximation; the dialog
// appends it to the acknowledgement shown after a successful commit. It is
// never itself a reason to refuse.
type PreparedSave struct {
	Paths, Existing []string
	Notice          string
	Commit          func(overwrite bool) ([]string, error)
}

type SaveDialogSeams struct {
	Directory     string
	List          func(directory string) (SaveDirectory, error)
	Prepare       func(SaveRequest) (PreparedSave, error)
	CanDelete     func(directory, name string) bool
	PrepareDelete func(directory, name string) (func() error, error)
}
