package nestedmixed

type DomainOrder struct {
	ID    string
	Lines []*DomainLine
}

type DomainLine struct {
	SKU      string
	Qty      int
	NoteText *string
}
