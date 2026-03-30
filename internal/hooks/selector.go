package hooks

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// Selector represents a parsed hook selector expression.
//
// Grammar (v1):
// - <selector> := <clause> ("&&" <clause>)*
// - <clause> := <key> ("==" | "!=") <value>
// - <key> uses [A-Z_][A-Z0-9_]*
// - <value> can be a bare token or a quoted string
//
// Matching semantics:
// - All clauses are ANDed
// - Missing keys in env never match (for both == and !=)
type Selector struct {
	clauses []selectorClause
}

type selectorClause struct {
	key   string
	value string
	op    selectorOp
}

type selectorOp string

const (
	selectorOpEqual    selectorOp = "=="
	selectorOpNotEqual selectorOp = "!="
)

// ParseSelector parses a hook selector expression.
// Empty expressions are valid and always match.
func ParseSelector(raw string) (Selector, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return Selector{}, nil
	}

	parts := strings.Split(trimmed, "&&")
	clauses := make([]selectorClause, 0, len(parts))

	for i, part := range parts {
		clauseText := strings.TrimSpace(part)
		if clauseText == "" {
			return Selector{}, fmt.Errorf("invalid selector: empty clause near position %d", selectorClauseStart(trimmed, i))
		}

		clause, err := parseSelectorClause(clauseText)
		if err != nil {
			return Selector{}, fmt.Errorf("invalid selector clause %q: %w", clauseText, err)
		}

		clauses = append(clauses, clause)
	}

	return Selector{clauses: clauses}, nil
}

func selectorClauseStart(raw string, clauseIndex int) int {
	if clauseIndex <= 0 {
		return 1
	}
	parts := strings.SplitN(raw, "&&", clauseIndex+1)
	if len(parts) <= clauseIndex {
		return len(raw) + 1
	}
	position := 1
	for i := 0; i < clauseIndex; i++ {
		position += len(parts[i]) + len("&&")
	}
	return position
}

func parseSelectorClause(input string) (selectorClause, error) {
	operatorIdx := strings.Index(input, "==")
	op := selectorOpEqual
	if operatorIdx == -1 {
		operatorIdx = strings.Index(input, "!=")
		op = selectorOpNotEqual
	}

	if operatorIdx == -1 {
		if strings.Contains(input, "=") {
			return selectorClause{}, fmt.Errorf("expected operator == or !=")
		}
		return selectorClause{}, fmt.Errorf("missing operator == or !=")
	}

	key := strings.TrimSpace(input[:operatorIdx])
	if key == "" {
		return selectorClause{}, fmt.Errorf("missing key")
	}
	if !isValidSelectorKey(key) {
		return selectorClause{}, fmt.Errorf("invalid key %q: expected [A-Z_][A-Z0-9_]*", key)
	}

	valueRaw := strings.TrimSpace(input[operatorIdx+len(op):])
	if valueRaw == "" {
		return selectorClause{}, fmt.Errorf("missing value")
	}

	value, err := parseSelectorValue(valueRaw)
	if err != nil {
		return selectorClause{}, err
	}

	return selectorClause{key: key, value: value, op: op}, nil
}

func isValidSelectorKey(key string) bool {
	if key == "" {
		return false
	}
	for i, r := range key {
		if i == 0 {
			if r != '_' && (r < 'A' || r > 'Z') {
				return false
			}
			continue
		}
		if r != '_' && !unicode.IsDigit(r) && (r < 'A' || r > 'Z') {
			return false
		}
	}
	return true
}

func parseSelectorValue(raw string) (string, error) {
	if strings.HasPrefix(raw, "\"") {
		unquoted, err := strconv.Unquote(raw)
		if err != nil {
			return "", fmt.Errorf("invalid quoted value: %w", err)
		}
		return unquoted, nil
	}

	if strings.ContainsAny(raw, " \t\n\r") {
		return "", fmt.Errorf("unquoted value cannot contain whitespace")
	}

	return raw, nil
}

// Match returns true when all selector clauses match the provided env map.
func (s Selector) Match(env map[string]string) bool {
	for _, clause := range s.clauses {
		envValue, ok := env[clause.key]
		if !ok {
			return false
		}

		switch clause.op {
		case selectorOpEqual:
			if envValue != clause.value {
				return false
			}
		case selectorOpNotEqual:
			if envValue == clause.value {
				return false
			}
		default:
			return false
		}
	}

	return true
}

// EvaluateSelector parses and evaluates a selector against the provided env map.
func EvaluateSelector(raw string, env map[string]string) (bool, error) {
	selector, err := ParseSelector(raw)
	if err != nil {
		return false, err
	}
	return selector.Match(env), nil
}
