package primitive_test

import (
	"caster-generator/primitive"
	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

type FooStringEnum string
type BarStringEnum string
type BuzzStringEnum string
type BellStringEnum string

func (FooStringEnum) IsValid() bool { panic("not implemented") }
func (BarStringEnum) IsValid() bool { panic("not implemented") }

func TestGenerateEnums(t *testing.T) {
	t.Parallel()

	t.Run("string enums", func(t *testing.T) {
		t.Parallel()

		res := primitive.Generate(reflect.TypeFor[FooStringEnum](), reflect.TypeFor[BarStringEnum](),
			"foo", "bar", "barStem", "ConvertFooToBar", primitive.CategoryAll)
		assert.NotEmpty(t, res)

		spew.Dump(res)
	})
}
