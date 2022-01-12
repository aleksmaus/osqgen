package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v2"
)

type Column struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
}

type Table struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Platforms   []string `json:"platforms"`
	Columns     []Column `json:"columns"`
}

type ColumnInfo struct {
	TableName string
	Column    Column
}

type ColumnDuplicate struct {
	Original  ColumnInfo
	Duplicate ColumnInfo
}

const (
	cmdFields = "fields"
	cmdReadme = "readme"
	cmdECS    = "ecs"
)

var (
	errUnsupportedCommand = errors.New("unsupported command")
)

var schemaFileName string

func exitOnError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
}

type ReaderCounter struct {
	io.ReadCloser
	count uint64
}

func Close(rc io.ReadCloser) {

}

func main() {
	var err error

	flag.Usage = func() {
		fmt.Println("Usage: osqgen [options] command")
		flag.PrintDefaults()
	}

	flag.StringVar(&schemaFileName, "schema", "", "Schema file name")

	flag.Parse()

	if schemaFileName == "" || len(flag.Args()) == 0 {
		flag.Usage()
		os.Exit(2)
	}

	command := flag.Args()[0]

	switch command {
	case cmdFields:
		err = execFieldsCommand()
	case cmdReadme:
		err = execReadmeCommand()
	case cmdECS:
		err = execECSCommand()
	default:
		err = errUnsupportedCommand
	}

	exitOnError(err)
}

func loadSchema(schemaFileName string) (tables []Table, err error) {
	files := strings.Split(schemaFileName, ",")

	for _, fn := range files {
		var t []Table
		t, err = loadSchemaFile(fn)
		if err != nil {
			return
		}
		tables = append(tables, t...)
	}
	return
}

func loadSchemaFile(schemaFileName string) (tables []Table, err error) {
	f, err := os.Open(schemaFileName)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	d := json.NewDecoder(f)
	err = d.Decode(&tables)
	return
}

func convergeToESType(t string) string {
	switch t {
	case "integer", "unsigned_bigint", "bigint":
		return "long"
	}
	return t
}

func printDuplicateColumns(dupColumnsMap map[string][]ColumnDuplicate) {

	keys := make([]string, 0, len(dupColumnsMap))
	for k := range dupColumnsMap {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, colName := range keys {
		dupColumns := dupColumnsMap[colName]
		c := len(dupColumns)
		for _, dup := range dupColumns {
			fmt.Printf("%d: %s.%s[%s] %s.%s[%s]\n",
				c,
				dup.Original.TableName, dup.Original.Column.Name, dup.Original.Column.Type,
				dup.Duplicate.TableName, dup.Duplicate.Column.Name, dup.Duplicate.Column.Type,
			)
		}
	}
}

func printColumnTypes(columns map[string]ColumnInfo) {
	types := make(map[string]struct{})
	for _, col := range columns {
		types[col.Column.Type] = struct{}{}
	}

	sorted := make([]string, 0, len(types))
	for k := range types {
		sorted = append(sorted, k)
	}

	sort.Strings(sorted)
	fmt.Println(sorted)
}

func execPreprocess() (columns map[string]ColumnInfo, dupColumnsMap map[string][]ColumnDuplicate, err error) {
	tables, err := loadSchema(schemaFileName)
	if err != nil {
		return
	}

	columns = make(map[string]ColumnInfo)

	dupColumnsMap = make(map[string][]ColumnDuplicate)

	total := 0
	for _, table := range tables {
		for _, column := range table.Columns {
			total += 1
			column.Type = convergeToESType(column.Type)
			column.Description = table.Name + `.` + column.Name + ` - ` + column.Description
			colInfo, ok := columns[column.Name]
			if ok {
				colInfo.Column.Description += "\n" + column.Description
				columns[column.Name] = colInfo

				if colInfo.Column.Type != column.Type {
					dupColumns, found := dupColumnsMap[column.Name]
					if !found {
						dupColumns = make([]ColumnDuplicate, 0)
					}
					dupColumns = append(dupColumns, ColumnDuplicate{
						Original:  colInfo,
						Duplicate: ColumnInfo{table.Name, column},
					})
					dupColumnsMap[column.Name] = dupColumns
				}
			} else {
				columns[column.Name] = ColumnInfo{table.Name, column}
			}
		}
	}
	return
}

func execFieldsCommand() error {
	columns, dupColumnsMap, err := execPreprocess()
	if err != nil {
		return err
	}
	return generateFields(os.Stdout, columns, dupColumnsMap)
}

func execReadmeCommand() error {
	columns, dupColumnsMap, err := execPreprocess()
	if err != nil {
		return err
	}
	return generateReadme(os.Stdout, columns, dupColumnsMap)
}

func generateReadme(w io.Writer, columns map[string]ColumnInfo, dupColumnsMap map[string][]ColumnDuplicate) error {

	var b strings.Builder

	if len(columns) > 0 {

		fields := make([]interface{}, 0, len(columns))

		sorted := make([]string, 0, len(fields))
		for k := range columns {
			sorted = append(sorted, k)
		}

		sort.Strings(sorted)

		for _, colName := range sorted {
			colInfo := columns[colName]
			types := []string{"keyword"}
			if colInfo.Column.Type == "text" {
				types = append(types, "text.text")
			} else {
				if _, ok := dupColumnsMap[colName]; !ok {
					types = append(types, "number."+colInfo.Column.Type)
				}
			}

			descriptions := strings.Split(colInfo.Column.Description, "\n")
			for i, description := range descriptions {
				b.WriteString(`| `)
				if i == 0 {
					b.WriteString(`**`)
					b.WriteString(colName)
					b.WriteString(`**`)
				}
				b.WriteString(` | `)
				pos := strings.Index(description, " - ")
				if pos != -1 {
					b.WriteString(`**`)
					b.WriteString(description[:pos])
					b.WriteString(`** - `)
					b.WriteString(description[pos+len(" - "):])
				} else {
					b.WriteString(description)
				}
				b.WriteString(` | `)
				if i == 0 {
					b.WriteString(strings.Join(types, ", "))
				}
				b.WriteString(" |\n")

			}
		}
	}
	fmt.Fprintln(w, b.String())
	return nil
}

func execECSCommand() error {
	f, err := os.Open(schemaFileName)
	if err != nil {
		return err
	}

	defer f.Close()

	d := yaml.NewDecoder(f)
	var root interface{}
	err = d.Decode(&root)
	if err != nil {
		return err
	}

	var ecsFields []string

	root = filter(root, "", &ecsFields)

	if len(ecsFields) > 0 {
		fmt.Println(`# This file is generated with osqgen (https://github.com/aleksmaus/osqgen) tool from the official ECS fields yml`)
		fmt.Println(`# To regenerate use:`)
		fmt.Println(`# osqgen --schema "./schema/ecs/fields.ecs_1.12.yml" ecs > ecs.yml`)
		for _, ecsField := range ecsFields {
			fmt.Println("- external: ecs")
			fmt.Println("  name:", ecsField)
		}
	}

	return nil
}

func filter(root interface{}, parent string, ecsFields *[]string) interface{} {
	nodes := root.([]interface{})

	var newNodes []interface{}
	for _, node := range nodes {
		m := node.(map[interface{}]interface{})

		childFields, ok := m["fields"]
		var name string
		if v, ok := m["name"]; ok {
			name = v.(string)
		}

		if ok {
			childFields = filter(childFields, joinPath(parent, name), ecsFields)
			if arr := childFields.([]interface{}); len(arr) == 0 {
				delete(m, "fields")
			} else {
				m["fields"] = childFields
			}
			newNodes = append(newNodes, node)
		} else if m["type"].(string) == "date" ||
			m["type"].(string) == "ip" ||
			m["type"].(string) == "long" ||
			m["type"].(string) == "float" ||
			m["type"].(string) == "boolean" {
			if name != "@timestamp" {
				*ecsFields = append(*ecsFields, joinPath(parent, name))
			}
			newNodes = append(newNodes, node)
		}
	}
	return newNodes
}

func joinPath(parent, name string) string {
	if parent != "" {
		return strings.Join([]string{parent, name}, ".")
	}
	return name
}
