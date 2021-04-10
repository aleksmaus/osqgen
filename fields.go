package main

import (
	"fmt"
	"io"

	"gopkg.in/yaml.v2"
)

type Node struct {
	Name        string        `yaml:"name,omitempty"`
	Title       string        `yaml:"title,omitempty"`
	Description string        `yaml:"description,omitempty"`
	Type        string        `yaml:"type,omitempty"`
	Fields      []interface{} `yaml:"fields,omitempty"`
}

type Field struct {
	Name        string     `yaml:"name,omitempty"`
	Description string     `yaml:"description,omitempty"`
	Type        string     `yaml:"type,omitempty"`
	IgnoreAbove int        `yaml:"ignore_above,omitempty"`
	Multifields []MulField `yaml:"multi_fields,omitempty"`
}

type MulField struct {
	Name         string `yaml:"name,omitempty"`
	Type         string `yaml:"type,omitempty"`
	Norms        bool   `yaml:"norms"`
	DefaultValue bool   `yaml:"default_field"`
}

// Following this recommendation for mapping
// https://www.elastic.co/guide/en/elasticsearch/reference/current/number.html
// Consider mapping a numeric identifier as a keyword if:

// You don’t plan to search for the identifier data using range queries.
// Fast retrieval is important. term query searches on keyword fields are often faster than term searches on numeric fields.
// If you’re unsure which to use, you can use a multi-field to map the data as both a keyword and a numeric data type.

// Most of the fields are mapped into keyword, the test fields also are mapped into the multifield test field
func generateFields(w io.Writer, columns map[string]ColumnInfo, dupColumnsMap map[string][]ColumnDuplicate) error {
	confs := []Node{
		{
			Name:        "osquery",
			Title:       "Osquery result",
			Description: "Fields related to the Osquery result",
			Type:        "group",
		},
	}

	conf := confs[0]

	if len(columns) > 0 {
		fields := make([]interface{}, 0, len(columns))

		for colName, colInfo := range columns {
			field := Field{
				Name:        colName,
				Description: colInfo.Column.Description,
				Type:        "keyword",
				IgnoreAbove: 1024,
			}
			if colInfo.Column.Type == "text" {
				field.Multifields = []MulField{
					{
						Name: "text",
						Type: "text",
					},
				}
			}
			fields = append(fields, field)
		}

		conf.Fields = fields
	}

	confs[0] = conf

	b, err := yaml.Marshal(confs)

	if err != nil {
		return err
	}

	fmt.Println(string(b))

	return nil
}

// - name: name
// level: extended
// type: keyword
// ignore_above: 1024
// multi_fields:
//   - name: text
// 	type: text
// 	norms: false
// 	default_field: false
// description: 'Process name.
