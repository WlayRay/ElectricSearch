package types

import (
	"strings"
)

func NewTermQuery(field, keyword string) *TermQuery {
	//TermQuery的一级成员里只有 Field-keyword非空，Must和Should都为空
	return &TermQuery{Keyword: &Keyword{Field: field, Word: keyword}}
}

func (tq TermQuery) Empty() bool {
	return tq.Keyword == nil && len(tq.Must) == 0 && len(tq.Should) == 0
}

// Builder模式
func (tq *TermQuery) And(querys ...*TermQuery) *TermQuery {
	if len(querys) == 0 {
		return tq
	}

	array := make([]*TermQuery, 0, len(querys)+1)
	if !tq.Empty() {
		array = append(array, tq)
	}

	for _, query := range querys {
		if !query.Empty() {
			array = append(array, query)
		}
	}
	return &TermQuery{Must: array}
}

func (tq *TermQuery) Or(querys ...*TermQuery) *TermQuery {
	if len(querys) == 0 {
		return tq
	}

	array := make([]*TermQuery, 0, len(querys)+1)
	if !tq.Empty() {
		array = append(array, tq)
	}

	for _, query := range querys {
		if !query.Empty() {
			array = append(array, query)
		}
	}
	return &TermQuery{Should: array}
}

func (tq TermQuery) ToString() string {
	if tq.Keyword != nil {
		return tq.Keyword.ToString()
	} else if len(tq.Must) > 0 {
		if len(tq.Must) == 1 {
			return tq.Must[0].ToString()
		} else {
			sb := strings.Builder{}
			sb.WriteByte('(')
			for _, e := range tq.Must {
				s := e.ToString()
				if len(s) > 0 {
					sb.WriteString(s)
					sb.WriteByte('&')
				}
			}
			s := sb.String()
			s = s[0:len(s)-1] + ")"
			return s
		}
	} else if len(tq.Should) > 0 {
		if len(tq.Should) == 1 {
			return tq.Should[0].ToString()
		} else {
			sb := strings.Builder{}
			sb.WriteByte('(')
			for _, e := range tq.Should {
				s := e.ToString()
				if len(s) > 0 {
					sb.WriteString(s)
					sb.WriteByte('|')
				}
			}
			s := sb.String()
			s = s[0:len(s)-1] + ")"
			return s
		}
	}
	return ""
}
