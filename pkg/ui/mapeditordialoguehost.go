package ui

// InspectionDialogues is implemented by edits whose host holds the map's
// interactive dialogues, one file each: it lists them, reads one, writes one
// with every condition and action node its replies name as one edit, removes
// one that no trigger uses, and says which triggers use one.
type InspectionDialogues interface {
	DialogueNames() []string
	ReadDialogue(name string) (DialogueData, error)
	// WriteDialogue writes the dialogue; a refusal names the check it broke.
	WriteDialogue(name string, d DialogueData) (*InspectionDocument, error)
	// WriteDialogueFor also writes the owner's trigger, whose planned action
	// starts the dialogue, in the same edit.
	WriteDialogueFor(owner InspectionMessageOwner, name string, d DialogueData) (*InspectionDocument, error)
	RemoveDialogue(name string) (*InspectionDocument, error)
	// DialogueUsers are the names of the triggers that start the dialogue.
	DialogueUsers(name string) []string
	// DialogueIssues are the file checks of d, each as a plain phrase.
	DialogueIssues(d DialogueData) []DialogueIssue
	// FreeMark is a script variable no node and no dialogue uses, -1 for none.
	FreeMark(d DialogueData) int
}

// validDialogueName reports that name can name a dialogue file: one to 32
// lower-case letters, digits or underscores.
func validDialogueName(name string) bool {
	if name == "" || len(name) > 32 {
		return false
	}
	for _, c := range name {
		if c != '_' && (c < 'a' || c > 'z') && (c < '0' || c > '9') {
			return false
		}
	}
	return true
}

// DialogueIssue is one broken check: Rule is the check's word key, Node the
// node's name, Reply its reply number from 1 (0 for the node), Other the
// next node's or variable's number.
type DialogueIssue struct {
	Rule  string
	Node  string
	Reply int
	Other int
}

// DialogueData is a dialogue as the host reads and writes it: nodes in file
// order, the first node starting.
type DialogueData struct {
	Nodes []DialogueNodeData
}

// DialogueNodeData is one node: its id, its name, its text by language code
// and its replies (at most four).
type DialogueNodeData struct {
	ID   int
	Name string
	Text map[string]string
	// Speaks says Speaker is the speaker record the page shows; Variants are
	// the page's texts by hero variant key, each by language; Next is where a
	// page with no replies goes on (0 ends).
	Speaks   bool
	Speaker  int
	Variants map[string]map[string]string
	Next     int
	Replies  []DialogueReplyData
}

// DialogueReplyData is one reply: its texts, the node it leads to (0 ends
// the dialogue), the once mark (0 for a reply that is not once, then Once is
// false), hidden or disabled, and its condition and outcome as the nodes
// the trigger wizard writes: Keep names a node as it stands.
type DialogueReplyData struct {
	Text     map[string]string
	Next     int
	Once     bool
	Mark     int
	Disabled bool
	Pairs    []WizardPair
	Actions  []WizardNode
}
