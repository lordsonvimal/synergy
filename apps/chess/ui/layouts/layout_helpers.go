package layouts

import "github.com/lordsonvimal/synergy/apps/chess/ui/helpers"

// datastarImportMapJSON returns the body of the page-level importmap that
// resolves the bare specifier "datastar" to the hashed runtime bundle. Lives
// in a .go file (not a .templ) so the JSON braces don't fight with templ's
// expression parser.
func datastarImportMapJSON() string {
	return `{"imports":{"datastar":"` + helpers.Asset("js/datastar.js") + `"}}`
}
