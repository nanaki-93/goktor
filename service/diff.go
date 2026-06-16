package service

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/nanaki-93/goktor/model"
)

const (
	DefaultDelimiter = '\t'
)

type DiffService interface {
	CalculateDiff(config model.DiffConfig) (string, error)
}
type DiffFileService struct {
	logger Logger
}

func NewDiffService() DiffService {
	return &DiffFileService{
		logger: &DefaultLogger{},
	}
}

func NewDiffServiceWithLogger(logger Logger) DiffService {
	return &DiffFileService{
		logger: logger,
	}
}

func (d *DiffFileService) CalculateDiff(config model.DiffConfig) (string, error) {

	leftMap, err := d.populateMap(config.Left, config)
	if err != nil {
		return "", fmt.Errorf("error populating left map: %s", err)
	}

	fmt.Println("Comparing content of lenght:" + strconv.Itoa(len(leftMap)))

	rightMap, err := d.populateMap(config.Right, config)
	if err != nil {
		return "", fmt.Errorf("error populating right map: %s", err)
	}

	fmt.Println("Comparing content of lenght:" + strconv.Itoa(len(leftMap)))

	resultMap, err := compare(leftMap, rightMap)
	if err != nil {
		return "", fmt.Errorf("error comparing files: %s", err)
	}

	var buf bytes.Buffer

	writer := csv.NewWriter(&buf)
	writer.Comma = delimiterOrDefault(config.Delimiter)

	for key, value := range resultMap {
		if err := writer.Write([]string{key, value}); err != nil {
			return "", fmt.Errorf("error writing csv row: %s", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", fmt.Errorf("error flushing csv writer: %s", err)
	}

	resultBytes := buf.Bytes()

	resultFile := fmt.Sprintf("%s.%s", config.Left, config.Right)
	if err := os.WriteFile(resultFile, resultBytes, 0644); err != nil {
		return "", fmt.Errorf("error writing result file: %s", err)
	}

	return resultFile, nil
}

func (d *DiffFileService) populateMap(fileName string, config model.DiffConfig) (map[string]model.DiffContent, error) {
	fileReader, err := getCsvReader(fileName, config.Delimiter)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %s", err)
	}

	rawContent, err := fileReader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("error reading file: %s", err)
	}
	contentMap := make(map[string]model.DiffContent)
	for i, value := range rawContent {
		if i == 0 && config.WithHeader {
			continue
		}
		if len(value) < 3 {
			return nil, fmt.Errorf("invalid row %d: expected at least 3 columns, got %d", i+1, len(value))
		}
		keyToAdd := value[0]
		_, exists := contentMap[value[0]]
		if exists {
			keyToAdd = keyToAdd + "-" + strconv.Itoa(i)
		}
		contentMap[keyToAdd] = model.DiffContent{Content: normalizeContentValue(value[1]), Type: value[2]}
	}
	return contentMap, nil
}

func normalizeContentValue(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 && value[0] == '\'' && value[len(value)-1] == '\'' {
		return value[1 : len(value)-1]
	}
	return value
}

func compare(content map[string]model.DiffContent, content2 map[string]model.DiffContent) (map[string]string, error) {
	result := make(map[string]string)
	fmt.Println("Comparing content of lenght:" + strconv.Itoa(len(content)))

	for key, value := range content {
		keyToSearch := strings.Split(key, "-")[0]
		keysToCompare := make([]string, 0)

		for key2, _ := range content2 {
			splittedKey := strings.Split(key2, "-")[0]
			if splittedKey == keyToSearch {
				keysToCompare = append(keysToCompare, key2)
			}
		}

		if len(keysToCompare) == 0 {
			result[key] = "MISSING"
			continue
		}

		result[key] = compareMultipleKeys(keysToCompare, content2, value)
	}
	return result, nil
}

func compareMultipleKeys(keysToCompare []string, content2 map[string]model.DiffContent, value model.DiffContent) string {
	for _, keyToCompare := range keysToCompare {
		valToCompare := content2[keyToCompare]
		isEqual, err := compareWithType(value, valToCompare)
		if err != nil {
			return fmt.Sprintf("error comparing values: %s", err)
		}

		if isEqual {
			return "OK"
		}

	}
	return "DIFF"

}

func compareWithType(left model.DiffContent, right model.DiffContent) (bool, error) {
	if strings.ToLower(left.Type) != strings.ToLower(right.Type) {
		return false, fmt.Errorf("types are not equal")
	}

	lowType := strings.ToLower(left.Type)
	switch {
	case lowType == model.JsonType:
		return jsonCompare(left, right)
	}

	if lowType == model.XmlType {
		return xmlCompare(left, right)
	}
	return reflect.DeepEqual(left.Content, right.Content), nil

}

func xmlCompare(left model.DiffContent, right model.DiffContent) (bool, error) {
	leftXml, err := normalizeXML(left.Content)
	if err != nil {
		return false, fmt.Errorf("error marshaling json: %s", err)
	}
	rightXml, err := normalizeXML(right.Content)
	if err != nil {
		return false, fmt.Errorf("error marshaling json: %s", err)
	}

	return leftXml == rightXml, nil
}

func jsonCompare(left model.DiffContent, right model.DiffContent) (bool, error) {
	var leftJson any
	var rightJson any
	err := json.Unmarshal([]byte(left.Content), &leftJson)
	if err != nil {
		return false, fmt.Errorf("error marshaling json: %s", err)
	}
	err = json.Unmarshal([]byte(right.Content), &rightJson)
	if err != nil {
		return false, fmt.Errorf("error marshaling json: %s", err)
	}
	return reflect.DeepEqual(leftJson, rightJson), nil
}
func normalizeXML(input string) (string, error) {
	decoder := xml.NewDecoder(strings.NewReader(input))

	var buf bytes.Buffer

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("invalid xml: %w", err)
		}

		switch t := token.(type) {
		case xml.StartElement:
			buf.WriteByte('<')
			writeXMLName(&buf, t.Name)

			sort.Slice(t.Attr, func(i, j int) bool {
				if t.Attr[i].Name.Space == t.Attr[j].Name.Space {
					return t.Attr[i].Name.Local < t.Attr[j].Name.Local
				}
				return t.Attr[i].Name.Space < t.Attr[j].Name.Space
			})

			for _, attr := range t.Attr {
				buf.WriteByte(' ')
				writeXMLName(&buf, attr.Name)
				buf.WriteByte('=')
				buf.WriteByte('"')
				xml.EscapeText(&buf, []byte(attr.Value))
				buf.WriteByte('"')
			}

			buf.WriteByte('>')

		case xml.EndElement:
			buf.WriteString("</")
			writeXMLName(&buf, t.Name)
			buf.WriteByte('>')

		case xml.CharData:
			text := string(t)
			if strings.TrimSpace(text) == "" {
				continue
			}
			xml.EscapeText(&buf, []byte(text))

		case xml.Comment:
			continue
		}
	}

	return buf.String(), nil
}

func writeXMLName(buf *bytes.Buffer, name xml.Name) {
	if name.Space != "" {
		buf.WriteByte('{')
		buf.WriteString(name.Space)
		buf.WriteByte('}')
	}

	buf.WriteString(name.Local)
}

func getCsvReader(path string, delim rune) (*csv.Reader, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %s", err)
	}
	reader := csv.NewReader(file)
	reader.Comma = delimiterOrDefault(delim)
	reader.LazyQuotes = true
	return reader, nil
}

func delimiterOrDefault(delimiter rune) rune {
	if delimiter == 0 {
		return DefaultDelimiter
	}
	return delimiter
}
