package forge

type Repo struct {
	Name           string
	Host           string
	WorkingDir     string
	VSCodeOpenArgs []string
}

func (r Repo) PkgGoDevURL() string {
	return "https://pkg.go.dev/" + r.Name
}

func (r Repo) URL() string {
	return "https://" + r.Name
}
