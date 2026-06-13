// Command gen generates a markdown provider table from the built-in
// provider registry. Run with: go run ./cmd/gen/
package main

import (
	"fmt"

	"github.com/dkmnx/kairo/internal/providers"
)

func main() {
	fmt.Println(providers.ProviderTableMarkdown())
}
