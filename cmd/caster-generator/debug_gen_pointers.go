//go:build ignore

package main

import (
	"fmt"
	"os"

	"caster-generator/internal/analyze"
	"caster-generator/internal/gen"
	"caster-generator/internal/mapping"
	"caster-generator/internal/plan"
)

func main() {
	m, err := mapping.LoadFromFile("./examples/pointers/map.yaml")
	if err != nil {
		fmt.Println("load mapping:", err)
		os.Exit(1)
	}

	gph, err := analyze.LoadPackages([]string{"caster-generator/examples/pointers"})
	if err != nil {
		fmt.Println("load packages:", err)
		os.Exit(1)
	}

	resolver := plan.NewResolver(gph, m, plan.DefaultConfig())
	p, err := resolver.Resolve()
	if err != nil {
		fmt.Println("resolve error:", err)
		fmt.Printf("diagnostics: %+v\n", p.Diagnostics)
		os.Exit(1)
	}
	if p.Diagnostics.HasErrors() {
		fmt.Println("resolve diagnostics:")
		fmt.Printf("%+v\n", p.Diagnostics)
		os.Exit(1)
	}

	generator := gen.NewGenerator(gen.DefaultGeneratorConfig())
	files, genErr := generator.Generate(p)
	if genErr != nil {
		fmt.Println("generate error:", genErr)
		for _, f := range files {
			fmt.Println("===", f.Filename, "===")
			fmt.Println(string(f.Content))
		}
		os.Exit(1)
	}

	for _, f := range files {
		fmt.Println("===", f.Filename, "===")
		fmt.Println(string(f.Content))
	}
}
