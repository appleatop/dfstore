package dbquery

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/bobbae/q"

	"go.mongodb.org/mongo-driver/bson"
)

// query session to convert condition and filter to bson.D for MongoDB queries
type dbquery struct {
	strPrefix  string                 // the global prefix of the replacing string
	mapCnt     int                    // the id for the ELEMxx replacing string
	mapElement map[string]interface{} // define the mapping of the ELEM key with the interface
}

// all string components for condition and filter will be converted into elements,
// the table name [[...]] is converted to elem_key
// the value field (convertVAlue(value[0].m)) {....} is converted to elem_value
// the operator which includes  ""== != < > <= >= IN BETWEEEN LIKE" is converted to elem_operator
// the aggregator which includes "AND OR" is converted to elem_aggregator
// Eventually, all these will be converted to elem_bson (bson.D)
type element interface{}

type elem_key struct {
	m string
}
type elem_value struct {
	m string
}

type elem_operator struct {
	m string
}

type elem_aggregator struct {
	m string // AND,  OR
}

type elem_composites struct {
	m composites
}

type elem_bson struct {
	m bson.D
}

type composites struct {
	elem []element
}

// these mappings are to avoid the regexp getting confused with special characters like { } [ ]
// which can be used in the user-given {value}. The algorithm will map those characters to some special
// replacement string and convert them back later when ele_value is constructed.

var mapStrValue map[string]string    // the mapping of special strings in {value}
var revMapStrValue map[string]string // the reverse mapping of special srings in {value}

func init() {
	mapStrValue = map[string]string{
		`(\\{)`:  `S1`,
		`(\\})`:  `S2`,
		`(\\\[)`: `S3`,
		`(\\\])`: `S4`,
	}
	revMapStrValue = map[string]string{
		`S1`: `\{`,
		`S2`: `\}`,
		`S3`: `\[`,
		`S4`: `\]`,
	}
}

// create new dbquery
// Pre-allocate memory for mapping
func New() dbquery {
	dq := dbquery{
		strPrefix: "@@",
		mapCnt:    0,
	}
	dq.mapElement = make(map[string]interface{})
	return dq
}

// create new composites
func newComposites() composites {
	comp := composites{}
	comp.elem = make([]element, 0)
	return comp
}

// Set the prefix of string to replace special strings.
// We have to ensure that this replacing string prefix does not exist in the original string
func (dq *dbquery) setMapStrPrefix(str string) {
	//q.Q("TRACE: ", str)
	dq.strPrefix = `@@`
	for {
		matched, _ := regexp.MatchString(dq.strPrefix, str)
		if matched != true {
			//q.Q("INFO: ", dq.strPrefix)
			return
		}
		dq.strPrefix = dq.strPrefix + `@`
	}
}

// To replace special strings as indicated in the map
func (dq *dbquery) replaceValueString(str string, m map[string]string) (string, error) {
	var instr, outstr string

	matched, _ := regexp.MatchString(dq.strPrefix, str)
	if matched {
		return str, fmt.Errorf("the replacing string prefix (%s) presented in the original string: %s", dq.strPrefix, str)
	}

	instr = str
	for key := range m {
		reg := regexp.MustCompile(key)
		outstr = reg.ReplaceAllString(instr, dq.strPrefix+m[key])
		instr = outstr
	}
	return outstr, nil
}

// To reverse back the replacement of the special string
func (dq *dbquery) revReplaceValueString(str string, m map[string]string) (string, error) {
	var instr, outstr string
	instr = str
	for key := range m {
		reg := regexp.MustCompile(dq.strPrefix + key)
		outstr = reg.ReplaceAllString(instr, m[key])
		instr = outstr
	}
	return outstr, nil
}

// Check if the input string has the valid pattern.
func (dq *dbquery) checkValidSyntax(inputstr string) error {
	// Currently, we don't allow {{ ... }}
	matched, _ := regexp.MatchString(`\{[\w]*\{`, inputstr)
	if matched {
		return fmt.Errorf("contain invalid string pattern ..{{, %s", inputstr)
	}
	// We don't allow [ x [ or ] x ] either
	matched, _ = regexp.MatchString(`\[\s*[\w]+\s*\[`, inputstr)
	if matched {
		return fmt.Errorf("contain invalid string pattern ..some string between two square open blankets, %s", inputstr)
	}
	matched, _ = regexp.MatchString(`\]\s*[\w]+\s*\]`, inputstr)
	if matched {
		return fmt.Errorf("contain invalid string pattern ..some string between two square close blankets, %s", inputstr)
	}
	return nil
}

// Contruct elem_value which is convertVAlue(value[0].m)s of comparators embedded inside {...}
// and return the converted string, which replaces elem_velue to ELEMxx
func (dq *dbquery) constructValueElem(inputstr string) (string, error) {

	// Pre-convert the special strings for easy regexp mapping
	str, err := dq.replaceValueString(inputstr, mapStrValue)
	if err != nil {
		return "", err
	}

	reg := regexp.MustCompile(`{[^\{\}]*}`)
	pos := reg.FindAllStringIndex(str, -1)

	str_index := 0
	var value_elem element
	var pos_start, pos_end int
	var valuestr, mapstr string
	var outstr string = ""
	for i := 0; i < len(pos); i++ {
		if len(pos[i]) != 2 {
			return outstr, fmt.Errorf("the matching {..} is missing: %s", inputstr)
		}
		pos_start = pos[i][0]
		pos_end = pos[i][1]
		outstr = outstr + str[str_index:pos_start]
		// get the value string; skip the `{` and `}` in the beginning and the ending
		valuestr = str[(pos_start + 1):(pos_end - 1)]
		// Post-convert the special strings back ;
		valuestr, err = dq.revReplaceValueString(valuestr, revMapStrValue)
		//q.Q("TRACE: ", pos_start, pos_end, str[str_index:pos_start], str[pos_start:pos_end], str[pos_end:], valuestr, err)
		if err != nil {
			return "", err
		}
		value_elem = elem_value{valuestr}
		mapstr = fmt.Sprintf("%sELEM%02d", dq.strPrefix, dq.mapCnt)
		dq.mapCnt++
		outstr = outstr + mapstr
		str_index = pos_end
		//q.Q("TRACE: ", value_elem, outstr, str_index)
		dq.mapElement[mapstr] = value_elem
	}
	if str_index != len(str) {
		outstr = outstr + str[str_index:]
	}
	//q.Q("TRACE: ", "output", outstr, dq.mapElement, dq.mapCnt)
	return outstr, nil
}

// Contruct elem_key which is the column names of the table embedded inside [[.. ]]
func (dq *dbquery) constructKeyElem(inputstr string) (string, error) {
	reg := regexp.MustCompile(`\[[\s]*\[[\w\.]*\][\s]*\]`)
	pos := reg.FindAllStringIndex(inputstr, -1)

	//q.Q("TRACE: ", pos)
	str_index := 0
	var key_elem element
	var pos_start, pos_end int
	var keystr, mapstr string
	var outstr string = ""
	for i := 0; i < len(pos); i++ {
		if len(pos[i]) != 2 {
			return outstr, fmt.Errorf("the matching [[..]] is missing: %s", inputstr)
		}
		pos_start = pos[i][0]
		pos_end = pos[i][1]
		outstr = outstr + inputstr[str_index:pos_start]
		keystr = inputstr[(pos_start):(pos_end)]
		// get the value string; skip the `[[` and `]]` in the beginning and the ending
		reg = regexp.MustCompile(`\[[\s]*\[`)
		keystr = reg.ReplaceAllString(keystr, "")
		reg = regexp.MustCompile(`\][\s]*\]`)
		keystr = reg.ReplaceAllString(keystr, "")
		//q.Q("TRACE: ", pos_start, pos_end, inputstr[pos_start:pos_end], keystr)
		key_elem = elem_key{keystr}
		mapstr = fmt.Sprintf("%sELEM%02d", dq.strPrefix, dq.mapCnt)
		dq.mapCnt++
		outstr = outstr + mapstr
		str_index = pos_end
		//q.Q("TRACE: ", key_elem, outstr, str_index)
		dq.mapElement[mapstr] = key_elem
	}
	if str_index != len(inputstr) {
		outstr = outstr + inputstr[str_index:]
	}
	//q.Q("TRACE: ", "output", outstr, dq.mapElement, dq.mapCnt)
	return outstr, nil
}

// convert the string to the mapping string and create either the elem_operator
// or the elem_aggregator. The mapping of the element and the mapping string will be
// stored in dq.mapElement. The function will return the mapping string.
// Note that this function assumes that all whitepsaces are already removed.
func (dq *dbquery) convertElemOpAggr(inputstr string) (string, error) {
	/* check for the aggregator */
	if (inputstr == `AND`) ||
		(inputstr == `OR`) {
		elem := elem_aggregator{inputstr}
		mapstr := fmt.Sprintf("%sELEM%02d", dq.strPrefix, dq.mapCnt)
		dq.mapCnt++
		dq.mapElement[mapstr] = elem
		//q.Q("TRACE: ", "element_aggregator", inputstr, mapstr, dq.mapElement[mapstr])
		return mapstr, nil
	}
	/* check for the operator */
	if (inputstr == `==`) ||
		(inputstr == `!=`) ||
		(inputstr == `<`) ||
		(inputstr == `>`) ||
		(inputstr == `IN`) {
		// TODO: Don't support BETWEEN yet; (inputstr == `BETWEEN`) {
		elem := elem_operator{inputstr}
		mapstr := fmt.Sprintf("%sELEM%02d", dq.strPrefix, dq.mapCnt)
		dq.mapCnt++
		dq.mapElement[mapstr] = elem
		//q.Q("TRACE: ", "element_operator", inputstr, mapstr, dq.mapElement[mapstr])
		return mapstr, nil
	}
	return "", fmt.Errorf("invalid operators and aggregators found (%s)", inputstr)
}

// Contruct elem_composites and replace the sring with the mapping string. dq.strPrefix+ELEMxx.
// This should be the string inside a pair of parentheses and this function assumes that all
// parentheses inside this are already converted to elem_composites. In addtions, the value
// and key are also converted to the elem_value and elem_key, respectively.
// This function will also create the elememts for operators and aggregators.
// Note that the returning mapping string, dq.strPrefix+ELEMxx will be used later to
// decode the group of elements inside a pair of parentheses in the upper hierarchy.
func (dq *dbquery) constructCompositesElem(inputstr string) (string, error) {
	new_elem_created := false
	comp := newComposites()
	// search for the operators and the aggregators and convert them to elem_operator and
	// elem_aggregators, respectively.

	// split the string by whitespaces to find all elements
	outstr := ""
	items := strings.Split(inputstr, " ")
	//q.Q("TRACE: ", items)
	for _, item := range items {
		//q.Q("TRACE: ", item)

		var elem element
		matched, _ := regexp.MatchString(`^`+dq.strPrefix, item)
		if matched {
			// this is an element, get its interface and link to the result
			elem = dq.mapElement[item]
			comp.elem = append(comp.elem, elem)
			outstr = outstr + item
		} else {
			// this should be either an operator or an aggregator,
			// remove all white spaces
			new_elem_created = true
			reg := regexp.MustCompile(`\s`)
			item = reg.ReplaceAllString(item, "")

			// convert it to elem and link to the result
			convstr, err := dq.convertElemOpAggr(item)
			if err != nil {
				return "", fmt.Errorf("contain the invalid operator/aggregator (%s) is invalid: %s", item, inputstr)
			}
			elem = dq.mapElement[convstr]
			outstr = outstr + convstr
			comp.elem = append(comp.elem, elem)
		}
	}
	// create the string mapping of this composites for further string parsing (upper hierarchy of groups)
	if new_elem_created {
		mapstr := fmt.Sprintf("%sELEM%02d", dq.strPrefix, dq.mapCnt)
		dq.mapCnt++
		dq.mapElement[mapstr] = elem_composites{comp}
		//q.Q("TRACE: ", "output composite", mapstr, dq.mapElement[mapstr], dq.mapCnt)
		return mapstr, nil
	}
	return inputstr, nil
}

// Translate the query to the group of elements (composites)
func (dq *dbquery) TranslateQuery(str string) (composites, error) {

	err := dq.checkValidSyntax(str)
	if err != nil {
		q.Q("ERROR: ", err)
		return composites{}, err
	}
	process_str, err := dq.constructValueElem(str)
	if err != nil {
		q.Q("ERROR: ", err)
		return composites{}, err
	}
	process_str, err = dq.constructKeyElem(process_str)
	if err != nil {
		q.Q("ERROR: ", err)
		return composites{}, err
	}
	//q.Q("TRACE: ", process_str)

	// break into groups of string inside and outside parentheses
	// elements inside the parenthesis will be grouped to composites.
	// for example ((A==5) AND (B!=3)) OR (C>0) will be decomposed to
	// composites, and contain the folowing elements
	// [1] elem_composites - ((A==5) AND (B!=3))
	// 	   [1.1] composites (A==5), elem_key(A), elem_operator(==), elem_value(5)
	//     [1.2] composites (B!=3), elem_key(B), elem_operator(!=), elem_value(3)
	// [2] elem_operator - OR, elem_aggregator(OR)
	// [3] elem_composites - (C>0), elem_key(C), elem_operator(>), elem_value(0)
	process_done := false
	for !process_done {
		// search for the most inner parentheses
		reg := regexp.MustCompile(`\([^\(\)]*\)`)
		pos := reg.FindAllStringIndex(process_str, -1)
		//q.Q("TRACE: ", "decompose group", process_str, pos)

		var pos_start, pos_end int
		str_index := 0
		outstr := ""
		for i := 0; i < len(pos); i++ {
			pos_start = pos[i][0]
			pos_end = pos[i][1]
			outstr = outstr + process_str[str_index:pos_start]
			// remove the parentheses at the beginning and the ending of the strings.
			groupstr := process_str[(pos_start + 1):(pos_end - 1)]
			//q.Q("TRACE: ", "group", pos_start, pos_end, process_str, groupstr)
			groupstr, err = dq.constructCompositesElem(groupstr)
			if err != nil {
				return composites{}, err
			}
			outstr = outstr + groupstr
			str_index = pos_end
		}

		if len(pos) != 0 {
			//q.Q("TRACE: ", "group done: ", process_str, outstr)
			outstr = outstr + process_str[pos_end:]
			process_str = outstr
		} else {
			// no more parentheses
			process_done = true
		}
	}

	// There may be some operators or aggregators left, convert them.
	//q.Q("TRACE: ", "last round process string: ", process_str)
	process_str, err = dq.constructCompositesElem(process_str)
	if err != nil {
		return composites{}, err
	}

	// There should be only one composite in the final result
	//q.Q("TRACE: ", "final process string: ", process_str)
	switch dq.mapElement[process_str].(type) {
	case elem_composites:
		comp := dq.mapElement[process_str].(elem_composites).m
		//q.Q("TRACE: ", "final result: ", process_str)
		//q.Q("INFO: ", "DBQuery:", str, comp, dq.mapElement)
		return comp, nil

	}
	return composites{}, fmt.Errorf("error in operation")
}

func convertValueArray(inputstr string) []string {
	// remove white space aroun ,
	reg := regexp.MustCompile(`[\s]*,[\s]*`)
	inputstr = reg.ReplaceAllString(inputstr, ",")
	// remove quote
	reg = regexp.MustCompile(`\"`)
	inputstr = reg.ReplaceAllString(inputstr, "")
	// Currently, we treat all values as string.
	result := strings.Split(inputstr, ",")
	//q.Q("TRACE: ", inputstr, result)
	return result
}

func convertValue(inputstr string) string {
	// Currently, we treat all values as string.
	// remove quote
	reg := regexp.MustCompile(`\"`)
	result := reg.ReplaceAllString(inputstr, "")
	// //q.Q("TRACE: ", inputstr, result)
	return result
}

func (dq *dbquery) constructBson(input composites) (bson.D, error) {
	// check the element
	var aggr elem_aggregator
	var operator elem_operator
	value := make([]elem_value, 0)
	key := make([]elem_key, 0)
	bson_comp := make([]bson.D, 0) // bson intepretation of the composites
	aggr_cnt := 0
	value_cnt := 0
	key_cnt := 0
	op_cnt := 0
	bsoncom_cnt := 0
	var elem interface{}

	for i := 0; i < len(input.elem); i++ {
		// composites must have the same aggregator
		elem = input.elem[i]
		switch elem.(type) {
		case elem_key:
			key = append(key, elem.(elem_key))
			key_cnt++
		case elem_operator:
			if op_cnt > 1 {
				return bson.D{}, fmt.Errorf("contain multiple operators in a group")
			}
			operator = elem.(elem_operator)
			op_cnt++
		case elem_aggregator:
			if (aggr_cnt > 1) && (aggr != elem) {
				return bson.D{}, fmt.Errorf("contain different operators in a group")
			}
			aggr = elem.(elem_aggregator)
			aggr_cnt++
		case elem_value:
			value = append(value, elem.(elem_value))
			value_cnt++
		case elem_composites:
			bson_composites, err := dq.constructBson(elem.(elem_composites).m)
			bsoncom_cnt++
			if err != nil {
				return bson.D{}, err
			}
			bson_comp = append(bson_comp, bson_composites)
		default:
			// do nothing
			q.Q("WARNING: ", i, "unrecognized interface", elem)
		}
	}
	// check if the convertVAlue(value[0].m) is the value or the composites
	//q.Q("TRACE: ", aggr_cnt, key_cnt, value_cnt, bsoncom_cnt, op_cnt)
	if (aggr_cnt == 0) && (key_cnt == 1) && (value_cnt == 1) {
		var dfilter bson.D
		switch operator.m {
		case "==":
			dfilter = bson.D{{key[0].m, convertValue(value[0].m)}}
		case "!=":
			dfilter = bson.D{{key[0].m, bson.D{{"$ne", convertValue(value[0].m)}}}}
		case ">":
			dfilter = bson.D{{key[0].m, bson.D{{"$gt", convertValue(value[0].m)}}}}
		case ">=":
			dfilter = bson.D{{key[0].m, bson.D{{"$gte", convertValue(value[0].m)}}}}
		case "<=":
			dfilter = bson.D{{key[0].m, bson.D{{"$lte", convertValue(value[0].m)}}}}
		case "<":
			dfilter = bson.D{{key[0].m, bson.D{{"$lt", convertValue(value[0].m)}}}}
		case "IN":
			//			dfilter = bson.D{{key[0].m, bson.D{{"$in", convertValueArray(value[0].m)}}}}
			dfilter = bson.D{{key[0].m, bson.D{{"$in", convertValueArray(value[0].m)}}}}
			// //q.Q("TRACE: ", dfilter)
		default:
			return bson.D{}, fmt.Errorf("invalid operator")
		}
		//q.Q("TRACE: ", dfilter)
		return dfilter, nil
	}
	//  shouldn't contain only composites now
	if (value_cnt != 0) || (key_cnt != 0) && (op_cnt != 0) {
		return bson.D{}, fmt.Errorf("contain invalid syntax in a group")
	}

	var afilter bson.A
	var qfilter bson.D

	for _, bson_elem := range bson_comp {
		afilter = append(afilter, bson_elem)
	}
	switch aggr.m {
	case "AND":
		qfilter = bson.D{{"$and", afilter}}
	case "OR":
		qfilter = bson.D{{"$or", afilter}}
	default:
		return bson.D{}, fmt.Errorf("invalid aggregator")
	}
	//q.Q("TRACE: ", qfilter)
	return qfilter, nil
}

// Translate the query to the group of elements (composites)
func (dq *dbquery) GetMongoQueryBson(str string) (bson.D, error) {
	var comp composites

	dq.setMapStrPrefix(str)

	// clear the element databases; use the new map
	dq.mapElement = make(map[string]interface{}, 60)
	dq.mapCnt = 0

	comp, err := dq.TranslateQuery(str)
	if err != nil {
		q.Q("ERROR: ", "translateQuery", err)
		return bson.D{}, err
	}

	return dq.constructBson(comp)
}
