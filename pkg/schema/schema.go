package schema

type Schema struct {
	Type    string     `yaml:"type"`
	Version string     `yaml:"version"`
	Spec    SchemaSpec `yaml:"spec"`
}

type SchemaSpec struct {
	Fields      []Field `yaml:"fields"`
	Compression string  `yaml:"compression"`
}

type Field struct {
	Name     string `yaml:"name"`
	Type     string `yaml:"type"`
	Encoding string `yaml:"encoding"`
}
