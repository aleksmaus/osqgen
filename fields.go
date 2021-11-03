package main

import (
	"fmt"
	"io"
	"sort"

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
	Name        string        `yaml:"name,omitempty"`
	Description string        `yaml:"description,omitempty"`
	Type        string        `yaml:"type,omitempty"`
	IgnoreAbove int           `yaml:"ignore_above,omitempty"`
	Multifields []interface{} `yaml:"multi_fields,omitempty"`
}

type MulField struct {
	Name         string `yaml:"name,omitempty"`
	Type         string `yaml:"type,omitempty"`
	Norms        bool   `yaml:"norms"`
	DefaultValue bool   `yaml:"default_field"`
}

type NumMulField struct {
	Name         string `yaml:"name,omitempty"`
	Type         string `yaml:"type,omitempty"`
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

		sorted := make([]string, 0, len(fields))
		for k := range columns {
			sorted = append(sorted, k)
		}

		sort.Strings(sorted)

		for _, colName := range sorted {
			colInfo := columns[colName]
			// Skip text columns altogether, rely on default
			// "dynamic_templates": [
			// 	{
			// 	  "strings_as_keyword": {
			// 		"match_mapping_type": "string",
			// 		"mapping": {
			// 		  "ignore_above": 1024,
			// 		  "type": "keyword"
			// 		}
			// 	  }
			// 	}
			// ]
			if colInfo.Column.Type == "text" {
				continue
			}
			field := Field{
				Name:        colName,
				Description: colInfo.Column.Description,
				Type:        "keyword",
				IgnoreAbove: 1024,
			}
			if colInfo.Column.Type == "text" {
				field.Multifields = []interface{}{
					MulField{
						Name: "text",
						Type: "text",
					},
				}
			} else {
				// add the actual type multifield if it's not a duplicate field with different types
				if _, ok := dupColumnsMap[colName]; !ok {
					field.Multifields = []interface{}{
						NumMulField{
							Name: "number",
							Type: colInfo.Column.Type,
						},
					}
				}
			}
			fields = append(fields, field)
		}

		conf.Fields = fields
	}

	confs[0] = conf

	yaml.FutureLineWrap()
	b, err := yaml.Marshal(confs)

	if err != nil {
		return err
	}

	fmt.Fprintln(w, string(b))
	return nil
}
