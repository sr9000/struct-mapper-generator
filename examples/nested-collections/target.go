package nestedcollections

type Target struct {
	Data map[string][]TargetInner
}
type TargetInner struct {
	Name string
}
