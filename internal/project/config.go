package project

type Config struct {
	Name    string
	Version string
}

type Project struct {
	Config
	Root string
}
