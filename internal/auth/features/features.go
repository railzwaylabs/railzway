package features

// ImplementedAuthFeatures is the source of truth for auth features available at runtime.
var ImplementedAuthFeatures = map[string]bool{
	"local":        true,
	"oauth":        true,
	"github":       false,
	"google":       false,
	"railzway_com": true,
}
