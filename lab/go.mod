module github.com/arahe-dev/hertasdk/lab

go 1.26.0

require (
	github.com/arahe-dev/hertasdk v0.0.0
	golang.org/x/sync v0.22.0
)

// The lab always measures the working tree, never a published version.
replace github.com/arahe-dev/hertasdk => ../
