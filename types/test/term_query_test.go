package termquerytest

import (
	"MiniES/types"
	"fmt"
	"testing"
)

const FIELD = ""

func TestTermQuery(t *testing.T) {
	A := types.NewTermQuery(FIELD, "") //空Expression
	B := types.NewTermQuery(FIELD, "B")
	C := types.NewTermQuery(FIELD, "C")
	D := types.NewTermQuery(FIELD, "D")
	E := &types.TermQuery{} //空Expression
	F := types.NewTermQuery(FIELD, "F")
	G := types.NewTermQuery(FIELD, "G")
	H := types.NewTermQuery(FIELD, "H")

	var q *types.TermQuery

	q = A
	fmt.Println(q.ToString())

	q = B.Or(C)
	fmt.Println(q.ToString())

	// (((((B)|C)&D)&(F|G))&H)
	q = A.Or(B).Or(C).And(D).Or(E).And(F.Or(G)).And(H)
	fmt.Println(q.ToString())

	A = types.NewTermQuery(FIELD, "A")
	E = types.NewTermQuery(FIELD, "E")

	// ((((((A|B)|C)&D)|E)&(F|G))&H)
	q = A.Or(B).Or(C).And(D).Or(E).And(F.Or(G)).And(H)
	fmt.Println(q.ToString())
}
