package mongoclient

import (
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"reflect"
	"strings"
)

/*
  All copied from https://github.com/mongodb/mongo-go-driver/blob/master/bson/bsoncodec/struct_tag_parser.go#L128
  As for now it is not available in the library
 */
var JSONFallbackStructTagParser bsoncodec.StructTagParserFunc = func(sf reflect.StructField) (bsoncodec.StructTags, error) {
	key := strings.ToLower(sf.Name)
	tag, ok := sf.Tag.Lookup("bson")
	if !ok {
		tag, ok = sf.Tag.Lookup("json")
	}
	if !ok && !strings.Contains(string(sf.Tag), ":") && len(sf.Tag) > 0 {
		tag = string(sf.Tag)
	}

	return parseTags(key, tag)
}

func parseTags(key string, tag string) (bsoncodec.StructTags, error) {
	var st bsoncodec.StructTags
	if tag == "-" {
		st.Skip = true
		return st, nil
	}

	for idx, str := range strings.Split(tag, ",") {
		if idx == 0 && str != "" {
			key = str
		}
		switch str {
		case "omitempty":
			st.OmitEmpty = true
		case "minsize":
			st.MinSize = true
		case "truncate":
			st.Truncate = true
		case "inline":
			st.Inline = true
		}
	}

	st.Name = key

	return st, nil
}