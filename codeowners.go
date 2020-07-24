package codeowners

// Ruleset is a slice of Rules
type Ruleset []Rule

// Match finds the last rule in the set that matches the path
func (r Ruleset) Match(path string) (*Rule, error) {
	for i := len(r) - 1; i >= 0; i-- {
		rule := r[i]
		match, err := rule.Match(path)
		if match || err != nil {
			return &rule, err
		}
	}
	return nil, nil
}

// Rule is a CODEOWNERS rule
type Rule struct {
	LineNumber int
	Pattern    pattern
	Owners     []Owner
	Comment    string
}

// Match tests whether path matches the rule's pattern
func (r Rule) Match(testPath string) (bool, error) {
	return r.Pattern.match(testPath)
}

// OwnerType is the type of file owner - one of 'email', 'team', or 'username
type OwnerType string

const (
	// OwnerTypeEmail is an owner type for email file owners
	OwnerTypeEmail OwnerType = "email"

	// OwnerTypeTeam is an owner type for GitHub team file owners
	OwnerTypeTeam OwnerType = "team"

	// OwnerTypeUsername is an owner type for GitHub username file owners
	OwnerTypeUsername OwnerType = "username"
)

// Owner represents a file owner
type Owner struct {
	Value string
	Type  OwnerType
}

// String returns a string representation of the owner
func (o Owner) String() string {
	if o.Type == "email" {
		return o.Value
	}
	return "@" + o.Value
}
