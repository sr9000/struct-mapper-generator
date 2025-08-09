package utils

func Second[T any](_ any, t T) T { return t }

func Unpack2[Slice ~[]T, T any](s Slice) (first T, second T) {
	switch len(s) {
	default:
		return s[0], s[1]
	case 0:
		return
	case 1:
		first = s[0]
		return
	}
}
