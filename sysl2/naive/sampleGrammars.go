package parser

import (
	sysl "github.com/anz-bank/sysl/sysl2/proto"
)

func makeQuantifierOptional() *sysl.Quantifier {
	return &sysl.Quantifier{Union: &sysl.Quantifier_Optional{}}
}

func makeQuantifierZeroPlus() *sysl.Quantifier {
	return &sysl.Quantifier{Union: &sysl.Quantifier_ZeroPlus{}}
}

func makeQuantifierOnePlus() *sysl.Quantifier {
	return &sysl.Quantifier{Union: &sysl.Quantifier_OnePlus{}}
}

func makeStringTerm(str string) *sysl.Term {
	return &sysl.Term{Atom: &sysl.Atom{Union: &sysl.Atom_String_{String_: str}}, Quantifier: nil}
}

func makeRegexpTerm(str string) *sysl.Term {
	return &sysl.Term{Atom: &sysl.Atom{Union: &sysl.Atom_Regexp{Regexp: str}}, Quantifier: nil}
}

func makeSequence(terms ...*sysl.Term) *sysl.Sequence {
	seq := sysl.Sequence{Term: terms}
	return &seq
}

func makeRule(name string) (*sysl.RuleName, *sysl.Term) {
	ruleName := sysl.RuleName{Name: name}
	ruleTerm := sysl.Term{Atom: &sysl.Atom{Union: &sysl.Atom_Rulename{Rulename: &ruleName}}, Quantifier: nil}
	return &ruleName, &ruleTerm
}

// S –> bab | bA
// A –> d | cA
func makeGrammar1() *sysl.Grammar {
	a := makeStringTerm("a")
	b := makeStringTerm("b")
	c := makeStringTerm("c")
	d := makeStringTerm("d")

	ruleNameA, A := makeRule("A")
	ruleNameS, _ := makeRule("S")

	return &sysl.Grammar{
		Name:  "test",
		Start: "S",
		Rules: map[string]*sysl.Rule{
			"S": {
				Name: ruleNameS,
				Choices: &sysl.Choice{
					Sequence: []*sysl.Sequence{
						makeSequence(b, a, b),
						makeSequence(b, A)},
				},
			},
			"A": {
				Name: ruleNameA,
				Choices: &sysl.Choice{
					Sequence: []*sysl.Sequence{
						makeSequence(d),
						makeSequence(c, A),
					},
				},
			},
		},
	}
}

// grammar := rule+
// rule := lhs ':' rhs ';'
// lhs := lowercaseName
// rhs := choice
// choice := seq ( '|' seq)*
// seq := term+
// term := atom quantifier?
// atom := STRING | ruleName | '(' choice  ')'
func makeEBNF() *sysl.Grammar {
	star := makeRegexpTerm("[*]")
	plus := makeRegexpTerm("[+]")
	qn := makeRegexpTerm("[?]")
	alt := makeRegexpTerm("[|]")
	colon := makeRegexpTerm("[:]")
	semiColon := makeRegexpTerm("[;]")
	openParen := makeRegexpTerm("[(]")
	closeParen := makeRegexpTerm("[)]")
	STRING := makeRegexpTerm(`['][^']*[']`)

	ruleNameRef := makeRegexpTerm("[a-zA-Z][0-9a-zA-Z_]*")

	lhsName, lhsTerm := makeRule("lhs")
	rhsName, rhsTerm := makeRule("rhs")
	ruleName, ruleTerm := makeRule("rule")
	grammarName, _ := makeRule("grammar")
	choiceName, choiceTerm := makeRule("choice")
	seqName, seqTerm := makeRule("seq")
	atomName, atomTerm := makeRule("atom")
	termName, termTerm := makeRule("term")
	termTerm.Quantifier = makeQuantifierOnePlus()
	quantifierName, quantifierTerm := makeRule("quantifier")
	quantifierTerm.Quantifier = makeQuantifierOptional()

	zeroPlusChoiceTerm := sysl.Term{
		Atom: &sysl.Atom{
			Union: &sysl.Atom_Choices{
				Choices: &sysl.Choice{
					Sequence: []*sysl.Sequence{
						makeSequence(alt, seqTerm),
					},
				},
			},
		},
		Quantifier: makeQuantifierZeroPlus(),
	}

	ruleTerm.Quantifier = makeQuantifierOnePlus()

	return &sysl.Grammar{
		Name:  "EBNF",
		Start: "grammar",
		Rules: map[string]*sysl.Rule{
			"grammar": {
				Name: grammarName,
				Choices: &sysl.Choice{
					Sequence: []*sysl.Sequence{
						makeSequence(ruleTerm),
					},
				},
			},
			"rule": {
				Name: ruleName,
				Choices: &sysl.Choice{
					Sequence: []*sysl.Sequence{
						makeSequence(lhsTerm, colon, rhsTerm, semiColon),
					},
				},
			},
			"lhs": {
				Name: lhsName,
				Choices: &sysl.Choice{
					Sequence: []*sysl.Sequence{
						makeSequence(ruleNameRef),
					},
				},
			},
			"rhs": {
				Name: rhsName,
				Choices: &sysl.Choice{
					Sequence: []*sysl.Sequence{
						makeSequence(choiceTerm),
					},
				},
			},
			"choice": {
				Name: choiceName,
				Choices: &sysl.Choice{
					Sequence: []*sysl.Sequence{
						makeSequence(seqTerm, &zeroPlusChoiceTerm),
					},
				},
			},
			"seq": {
				Name: seqName,
				Choices: &sysl.Choice{
					Sequence: []*sysl.Sequence{
						makeSequence(termTerm),
					},
				},
			},
			"term": {
				Name: termName,
				Choices: &sysl.Choice{
					Sequence: []*sysl.Sequence{
						makeSequence(atomTerm, quantifierTerm),
					},
				},
			},
			"atom": {
				Name: atomName,
				Choices: &sysl.Choice{
					Sequence: []*sysl.Sequence{
						makeSequence(STRING),
						makeSequence(ruleNameRef),
						makeSequence(openParen, choiceTerm, closeParen),
					},
				},
			},
			"quantifier": {
				Name: quantifierName,
				Choices: &sysl.Choice{
					Sequence: []*sysl.Sequence{
						makeSequence(star),
						makeSequence(plus),
						makeSequence(qn),
					},
				},
			},
		}}
}

// E  -> T E'
// E' -> + T E' | -TE' |epsilon
// T  -> F T'
// T' -> * F T' | /FT' |epsilon
// F  -> (E) | int
func makeEXPR() *sysl.Grammar {
	plus := makeRegexpTerm("[+]")
	minus := makeRegexpTerm("[-]")
	star := makeRegexpTerm("[*]")
	divide := makeRegexpTerm("[/]")
	openParen := makeRegexpTerm("[(]")
	closeParen := makeRegexpTerm("[)]")
	integer := makeRegexpTerm("[0-9]+")

	ERuleName, ETerm := makeRule("E")
	ETailRuleName, ETailTerm := makeRule("ETail")
	TRuleName, TTerm := makeRule("T")
	TTailRuleName, TTailTerm := makeRule("TTail")
	factorRuleName, factorTerm := makeRule("factor")

	return &sysl.Grammar{
		Name:  "EXPR",
		Start: "E",
		Rules: map[string]*sysl.Rule{
			"E": {
				Name: ERuleName,
				Choices: &sysl.Choice{
					Sequence: []*sysl.Sequence{
						makeSequence(TTerm, ETailTerm),
					},
				},
			},
			"ETail": {
				Name: ETailRuleName,
				Choices: &sysl.Choice{
					Sequence: []*sysl.Sequence{
						makeSequence(plus, TTerm, ETailTerm),
						makeSequence(minus, TTerm, ETailTerm),
						makeSequence(nil),
					},
				},
			},
			"T": {
				Name: TRuleName,
				Choices: &sysl.Choice{
					Sequence: []*sysl.Sequence{
						makeSequence(factorTerm, TTailTerm),
					},
				},
			},
			"TTail": {
				Name: TTailRuleName,
				Choices: &sysl.Choice{
					Sequence: []*sysl.Sequence{
						makeSequence(star, factorTerm, TTailTerm),
						makeSequence(divide, factorTerm, TTailTerm),
						makeSequence(nil),
					},
				},
			},
			"factor": {
				Name: factorRuleName,
				Choices: &sysl.Choice{
					Sequence: []*sysl.Sequence{
						makeSequence(openParen, ETerm, closeParen),
						makeSequence(integer),
					},
				},
			},
		},
	}
}

// obj
//    : '{' number (',' number)* '}'
//    | '{' '}'
//    ;
func makeRepeatSeq(quantifier *sysl.Quantifier) *sysl.Grammar {
	curlyOpen := makeRegexpTerm("[{]")
	curlyClosed := makeRegexpTerm("[}]")
	comma := makeRegexpTerm("[,]")
	number := makeRegexpTerm("[0-9]+")

	objRuleName, _ := makeRule("obj")
	obj2RuleName, obj2Term := makeRule("obj2")
	obj2Term.Quantifier = quantifier

	return &sysl.Grammar{
		Name:  "array",
		Start: "obj",
		Rules: map[string]*sysl.Rule{
			"obj": {
				Name: objRuleName,
				Choices: &sysl.Choice{
					Sequence: []*sysl.Sequence{
						makeSequence(curlyOpen, number, obj2Term, curlyClosed),
						makeSequence(curlyOpen, curlyClosed),
					},
				},
			},
			"obj2": {
				Name: obj2RuleName,
				Choices: &sysl.Choice{
					Sequence: []*sysl.Sequence{
						makeSequence(comma, number),
					},
				},
			},
		},
	}
}

// json
//    : value
//    ;

// obj
//    : '{' pair (',' pair)* '}'
//    | '{' '}'
//    ;

// pair
//    : STRING ':' value
//    ;

// array
//    : '[' value (',' value)* ']'
//    | '[' ']'
//    ;

// value
//    : STRING
//    | NUMBER
//    | obj
//    | array
func makeJSON(quantifier *sysl.Quantifier) *sysl.Grammar {
	// doubleQuote := makeStringTerm("\"")
	// singleQuote := makeStringTerm("'")
	curlyOpen := makeRegexpTerm("[{]")
	curlyClosed := makeRegexpTerm("[}]")
	comma := makeRegexpTerm("[,]")
	sqOpen := makeRegexpTerm("[[]")
	sqClose := makeRegexpTerm("[]]")
	colon := makeRegexpTerm("[:]")
	number := makeRegexpTerm("[0-9]+")
	STRING := makeRegexpTerm(`["][^"]*["]`)

	jsonRuleName, _ := makeRule("json")
	valueRuleName, valueTerm := makeRule("value")
	objRuleName, objTerm := makeRule("obj")
	pairRuleName, pairTerm := makeRule("pair")
	arrayRuleName, arrayTerm := makeRule("array")

	obj2Term := sysl.Term{
		Atom: &sysl.Atom{
			Union: &sysl.Atom_Choices{
				Choices: &sysl.Choice{
					Sequence: []*sysl.Sequence{
						makeSequence(comma, pairTerm),
					},
				},
			},
		},
		Quantifier: quantifier,
	}

	array2Term := sysl.Term{
		Atom: &sysl.Atom{
			Union: &sysl.Atom_Choices{
				Choices: &sysl.Choice{
					Sequence: []*sysl.Sequence{
						makeSequence(comma, valueTerm),
					},
				},
			},
		},
		Quantifier: quantifier,
	}

	return &sysl.Grammar{
		Name:  "json",
		Start: "json",
		Rules: map[string]*sysl.Rule{
			"obj": {
				Name: objRuleName,
				Choices: &sysl.Choice{
					Sequence: []*sysl.Sequence{
						makeSequence(curlyOpen, pairTerm, &obj2Term, curlyClosed),
						makeSequence(curlyOpen, curlyClosed),
					},
				},
			},
			"value": {
				Name: valueRuleName,
				Choices: &sysl.Choice{
					Sequence: []*sysl.Sequence{
						makeSequence(STRING),
						makeSequence(number),
						makeSequence(objTerm),
						makeSequence(arrayTerm),
					},
				},
			},
			"json": {
				Name: jsonRuleName,
				Choices: &sysl.Choice{
					Sequence: []*sysl.Sequence{
						makeSequence(valueTerm),
					},
				},
			},
			"pair": {
				Name: pairRuleName,
				Choices: &sysl.Choice{
					Sequence: []*sysl.Sequence{
						makeSequence(STRING, colon, valueTerm),
					},
				},
			},
			"array": {
				Name: arrayRuleName,
				Choices: &sysl.Choice{
					Sequence: []*sysl.Sequence{
						makeSequence(sqOpen, valueTerm, &array2Term, sqClose),
						makeSequence(sqOpen, sqClose),
					},
				},
			},
		},
	}
}

func makeG2() *sysl.Grammar {
	a := makeStringTerm("a")
	b := makeStringTerm("b")
	d := makeStringTerm("d")

	SruleName, _ := makeRule("S")
	AruleName, ATerm := makeRule("A")
	BruleName, BTerm := makeRule("B")
	DruleName, DTerm := makeRule("D")

	return &sysl.Grammar{
		Name:  "G2",
		Start: "S",
		Rules: map[string]*sysl.Rule{
			"S": {
				Name: SruleName,
				Choices: &sysl.Choice{
					Sequence: []*sysl.Sequence{
						makeSequence(ATerm, a),
					},
				},
			},
			"A": {
				Name: AruleName,
				Choices: &sysl.Choice{
					Sequence: []*sysl.Sequence{
						makeSequence(BTerm, DTerm),
					},
				},
			},
			"B": {
				Name: BruleName,
				Choices: &sysl.Choice{
					Sequence: []*sysl.Sequence{
						makeSequence(b), makeSequence(nil),
					},
				},
			},
			"D": {
				Name: DruleName,
				Choices: &sysl.Choice{
					Sequence: []*sysl.Sequence{
						makeSequence(d), makeSequence(nil),
					},
				},
			},
		},
	}
}

func makeNestedGrammar() *sysl.Grammar {
	a := makeRegexpTerm("{[A-Za-z]+:")
	b := makeStringTerm(":}")
	SruleName, _ := makeRule("S")

	return &sysl.Grammar{
		Name:  "G2",
		Start: "S",
		Rules: map[string]*sysl.Rule{
			"S": {
				Name: SruleName,
				Choices: &sysl.Choice{
					Sequence: []*sysl.Sequence{
						makeSequence(a),
						makeSequence(b),
					},
				},
			},
		},
	}
}

// func main() {
// 	fmt.Println("parsing grammar")

// }
