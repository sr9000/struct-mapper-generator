package nestedcollections

type Source struct {
	Data map[string][]SourceInner
}
type SourceInner struct {
	Name string
}
