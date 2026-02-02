package virtual

// Source is the only type in this example, containing various pointer fields.
// It is designed to test generation target type "on-the-fly" with pointer-heavy shapes.
type Source struct {
	ID        string
	Name      *string
	Items     []*SourceItem
	ExtraInfo *string
}

type SourceItem struct {
	ProductID string
	Quantity  *int
}
