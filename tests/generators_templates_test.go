package tests

import (
	"testing"

	"Game-Developers-World/seed/internal/generators/templates"
)

func TestGeneratorsTemplatesLoad(t *testing.T) {
	t.Run("loads component template", func(t *testing.T) {
		tpl, err := templates.Load("component")
		if err != nil {
			t.Fatal(err)
		}
		if !contains(tpl, "struct {{.Name}}") {
			t.Fatalf("expected component template to contain 'struct {{.Name}}', got: %s", tpl)
		}
		if !contains(tpl, "size_of") {
			t.Fatalf("expected component template to contain 'size_of', got: %s", tpl)
		}
		if !contains(tpl, "builder.has") {
			t.Fatalf("expected component template to contain 'builder.has', got: %s", tpl)
		}
	})

	t.Run("loads trait template", func(t *testing.T) {
		tpl, err := templates.Load("trait")
		if err != nil {
			t.Fatal(err)
		}
		if !contains(tpl, "struct {{.Name}}") {
			t.Fatalf("expected trait template to contain 'struct {{.Name}}', got: %s", tpl)
		}
	})

	t.Run("loads entity template", func(t *testing.T) {
		tpl, err := templates.Load("entity")
		if err != nil {
			t.Fatal(err)
		}
		if !contains(tpl, "struct {{.Name}}") {
			t.Fatalf("expected entity template to contain 'struct {{.Name}}', got: %s", tpl)
		}
	})

	t.Run("loads archetype template", func(t *testing.T) {
		tpl, err := templates.Load("archetype")
		if err != nil {
			t.Fatal(err)
		}
		if !contains(tpl, "struct {{.Name}}") {
			t.Fatalf("expected archetype template to contain 'struct {{.Name}}', got: %s", tpl)
		}
	})

	t.Run("loads state_machine template", func(t *testing.T) {
		tpl, err := templates.Load("state_machine")
		if err != nil {
			t.Fatal(err)
		}
		if !contains(tpl, "register_controller") {
			t.Fatalf("expected state_machine template to contain 'register_controller', got: %s", tpl)
		}
		if !contains(tpl, "add_transition") {
			t.Fatalf("expected state_machine template to contain 'add_transition', got: %s", tpl)
		}
		if !contains(tpl, "Fsm<std::string, std::string>") {
			t.Fatalf("expected state_machine template to contain 'Fsm<std::string, std::string>', got: %s", tpl)
		}
	})

	t.Run("loads system template", func(t *testing.T) {
		tpl, err := templates.Load("system")
		if err != nil {
			t.Fatal(err)
		}
		if !contains(tpl, "CommandProducer") {
			t.Fatalf("expected system template to contain 'CommandProducer', got: %s", tpl)
		}
		if !contains(tpl, "EphemeralProducer") {
			t.Fatalf("expected system template to contain 'EphemeralProducer', got: %s", tpl)
		}
		if !contains(tpl, "register_controller") {
			t.Fatalf("expected system template to contain 'register_controller', got: %s", tpl)
		}
	})

	t.Run("returns error for unknown kind", func(t *testing.T) {
		_, err := templates.Load("unknown")
		if err == nil {
			t.Fatal("expected error for unknown template kind")
		}
	})
}

func TestGeneratorsTemplatesLoadCpp(t *testing.T) {
	t.Run("loads system .cpp template", func(t *testing.T) {
		tpl, err := templates.LoadCpp("system")
		if err != nil {
			t.Fatal(err)
		}
		if !contains(tpl, "operator()") {
			t.Fatalf("expected system .cpp template to contain 'operator()', got: %s", tpl)
		}
		if !contains(tpl, "field_span") {
			t.Fatalf("expected system .cpp template to contain field_span comment, got: %s", tpl)
		}
	})

	t.Run("returns error for unknown kind", func(t *testing.T) {
		_, err := templates.LoadCpp("component")
		if err == nil {
			t.Fatal("expected error for unknown cpp template kind")
		}
	})
}
