package taggedurn

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaggedUrnCreation(t *testing.T) {
	taggedUrn, err := NewTaggedUrnFromString("cap:op=transform;format=json;data_processing")

	assert.NoError(t, err)
	assert.NotNil(t, taggedUrn)
	assert.Equal(t, "cap", taggedUrn.GetPrefix())

	// data_processing is a valueless tag, stored as * (must-have-any)
	dataProcessing, exists := taggedUrn.GetTag("data_processing")
	assert.True(t, exists)
	assert.Equal(t, "*", dataProcessing)

	op, exists := taggedUrn.GetTag("op")
	assert.True(t, exists)
	assert.Equal(t, "transform", op)

	format, exists := taggedUrn.GetTag("format")
	assert.True(t, exists)
	assert.Equal(t, "json", format)
}

func TestCustomPrefix(t *testing.T) {
	urn, err := NewTaggedUrnFromString("myapp:op=generate;ext=pdf")
	require.NoError(t, err)

	assert.Equal(t, "myapp", urn.GetPrefix())
	op, exists := urn.GetTag("op")
	assert.True(t, exists)
	assert.Equal(t, "generate", op)
	assert.Equal(t, "myapp:ext=pdf;op=generate", urn.ToString())
}

func TestPrefixCaseInsensitive(t *testing.T) {
	urn1, err := NewTaggedUrnFromString("CAP:op=test")
	require.NoError(t, err)
	urn2, err := NewTaggedUrnFromString("cap:op=test")
	require.NoError(t, err)
	urn3, err := NewTaggedUrnFromString("Cap:op=test")
	require.NoError(t, err)

	assert.Equal(t, "cap", urn1.GetPrefix())
	assert.Equal(t, "cap", urn2.GetPrefix())
	assert.Equal(t, "cap", urn3.GetPrefix())
	assert.True(t, urn1.Equals(urn2))
	assert.True(t, urn2.Equals(urn3))
}

func TestPrefixMismatchError(t *testing.T) {
	urn1, err := NewTaggedUrnFromString("cap:op=test")
	require.NoError(t, err)
	urn2, err := NewTaggedUrnFromString("myapp:op=test")
	require.NoError(t, err)

	_, err = urn1.Matches(urn2)
	assert.Error(t, err)
	capError, ok := err.(*TaggedUrnError)
	assert.True(t, ok)
	assert.Equal(t, ErrorPrefixMismatch, capError.Code)
}

func TestBuilderWithPrefix(t *testing.T) {
	urn, err := NewTaggedUrnBuilder("custom").
		Tag("key", "value").
		Build()
	require.NoError(t, err)

	assert.Equal(t, "custom", urn.GetPrefix())
	assert.Equal(t, "custom:key=value", urn.ToString())
}

func TestCanonicalStringFormat(t *testing.T) {
	taggedUrn, err := NewTaggedUrnFromString("cap:op=generate;target=thumbnail;ext=pdf")
	require.NoError(t, err)

	// Should be sorted alphabetically and have no trailing semicolon in canonical form
	assert.Equal(t, "cap:ext=pdf;op=generate;target=thumbnail", taggedUrn.ToString())
}

func TestPrefixRequired(t *testing.T) {
	// Missing prefix should fail
	taggedUrn, err := NewTaggedUrnFromString("op=generate;ext=pdf")
	assert.Nil(t, taggedUrn)
	assert.Error(t, err)
	assert.Equal(t, ErrorMissingPrefix, err.(*TaggedUrnError).Code)

	// Empty prefix should fail
	taggedUrn, err = NewTaggedUrnFromString(":op=generate")
	assert.Nil(t, taggedUrn)
	assert.Error(t, err)
	assert.Equal(t, ErrorEmptyPrefix, err.(*TaggedUrnError).Code)

	// Valid prefix should work
	taggedUrn, err = NewTaggedUrnFromString("cap:op=generate;ext=pdf")
	assert.NoError(t, err)
	assert.NotNil(t, taggedUrn)
	op, exists := taggedUrn.GetTag("op")
	assert.True(t, exists)
	assert.Equal(t, "generate", op)

	// Case-insensitive prefix
	taggedUrn, err = NewTaggedUrnFromString("CAP:op=generate")
	assert.NoError(t, err)
	op, exists = taggedUrn.GetTag("op")
	assert.True(t, exists)
	assert.Equal(t, "generate", op)
}

func TestTrailingSemicolonEquivalence(t *testing.T) {
	// Both with and without trailing semicolon should be equivalent
	urn1, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf")
	require.NoError(t, err)

	urn2, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf;")
	require.NoError(t, err)

	// They should be equal
	assert.True(t, urn1.Equals(urn2))

	// They should have same hash
	assert.Equal(t, urn1.Hash(), urn2.Hash())

	// They should have same string representation (canonical form)
	assert.Equal(t, urn1.ToString(), urn2.ToString())

	// They should match each other
	matches1, err := urn1.Matches(urn2)
	require.NoError(t, err)
	assert.True(t, matches1)

	matches2, err := urn2.Matches(urn1)
	require.NoError(t, err)
	assert.True(t, matches2)
}

func TestInvalidTaggedUrn(t *testing.T) {
	taggedUrn, err := NewTaggedUrnFromString("")

	assert.Nil(t, taggedUrn)
	assert.Error(t, err)
	assert.Equal(t, ErrorInvalidFormat, err.(*TaggedUrnError).Code)
}

func TestValuelessTagParsing(t *testing.T) {
	// Value-less tag is now valid and treated as wildcard
	taggedUrn, err := NewTaggedUrnFromString("cap:optimize")

	assert.NotNil(t, taggedUrn)
	assert.NoError(t, err)
	value, exists := taggedUrn.GetTag("optimize")
	assert.True(t, exists)
	assert.Equal(t, "*", value)
	assert.Equal(t, "cap:optimize", taggedUrn.ToString())
}

func TestInvalidCharacters(t *testing.T) {
	taggedUrn, err := NewTaggedUrnFromString("cap:type@invalid=value")

	assert.Nil(t, taggedUrn)
	assert.Error(t, err)
	assert.Equal(t, ErrorInvalidCharacter, err.(*TaggedUrnError).Code)
}

func TestTagMatching(t *testing.T) {
	urn, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf;target=thumbnail")
	require.NoError(t, err)

	// Exact match
	request1, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf;target=thumbnail")
	require.NoError(t, err)
	matches, err := urn.Matches(request1)
	require.NoError(t, err)
	assert.True(t, matches)

	// Subset match
	request2, err := NewTaggedUrnFromString("cap:op=generate")
	require.NoError(t, err)
	matches, err = urn.Matches(request2)
	require.NoError(t, err)
	assert.True(t, matches)

	// Wildcard match
	request3, err := NewTaggedUrnFromString("cap:ext=*")
	require.NoError(t, err)
	matches, err = urn.Matches(request3)
	require.NoError(t, err)
	assert.True(t, matches)

	// No match - conflicting value
	request4, err := NewTaggedUrnFromString("cap:op=extract")
	require.NoError(t, err)
	matches, err = urn.Matches(request4)
	require.NoError(t, err)
	assert.False(t, matches)
}

func TestMissingTagHandling(t *testing.T) {
	// NEW SEMANTICS: Missing tag in instance means the tag doesn't exist.
	// Pattern constraints must be satisfied by instance.

	instance, err := NewTaggedUrnFromString("cap:op=generate")
	require.NoError(t, err)

	// Pattern with tag that instance doesn't have: NO MATCH
	// Pattern ext=pdf requires instance to have ext=pdf, but instance doesn't have ext
	pattern1, err := NewTaggedUrnFromString("cap:ext=pdf")
	require.NoError(t, err)
	matches, err := instance.Matches(pattern1)
	require.NoError(t, err)
	assert.False(t, matches) // Instance missing ext, pattern wants ext=pdf

	// Pattern missing tag = no constraint: MATCH
	// Instance has op=generate, pattern has no constraint on op
	instance2, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf")
	require.NoError(t, err)
	pattern2, err := NewTaggedUrnFromString("cap:op=generate")
	require.NoError(t, err)
	matches, err = instance2.Matches(pattern2)
	require.NoError(t, err)
	assert.True(t, matches) // Instance has ext=pdf, pattern doesn't constrain ext

	// To match any value of a tag, use explicit ? or *
	pattern3, err := NewTaggedUrnFromString("cap:ext=?")
	require.NoError(t, err)
	matches, err = instance.Matches(pattern3)
	require.NoError(t, err)
	assert.True(t, matches) // Instance missing ext, pattern doesn't care

	// * means must-have-any - instance must have the tag
	pattern4, err := NewTaggedUrnFromString("cap:ext=*")
	require.NoError(t, err)
	matches, err = instance.Matches(pattern4)
	require.NoError(t, err)
	assert.False(t, matches) // Instance missing ext, pattern requires ext to be present
}

func TestSpecificity(t *testing.T) {
	// NEW GRADED SPECIFICITY:
	// K=v (exact value): 3 points
	// K=* (must-have-any): 2 points
	// K=! (must-not-have): 1 point
	// K=? (unspecified): 0 points

	urn1, err := NewTaggedUrnFromString("cap:op=*") // * = 2 points
	require.NoError(t, err)

	urn2, err := NewTaggedUrnFromString("cap:op=generate") // exact = 3 points
	require.NoError(t, err)

	urn3, err := NewTaggedUrnFromString("cap:op=*;ext=pdf") // * + exact = 2 + 3 = 5 points
	require.NoError(t, err)

	urn4, err := NewTaggedUrnFromString("cap:op=?") // ? = 0 points
	require.NoError(t, err)

	urn5, err := NewTaggedUrnFromString("cap:op=!") // ! = 1 point
	require.NoError(t, err)

	assert.Equal(t, 2, urn1.Specificity()) // * = 2
	assert.Equal(t, 3, urn2.Specificity()) // exact = 3
	assert.Equal(t, 5, urn3.Specificity()) // * + exact = 2 + 3
	assert.Equal(t, 0, urn4.Specificity()) // ? = 0
	assert.Equal(t, 1, urn5.Specificity()) // ! = 1

	// Specificity tuple for tie-breaking: (exact_count, must_have_any_count, must_not_count)
	exact, mustHaveAny, mustNot := urn2.SpecificityTuple()
	assert.Equal(t, 1, exact)
	assert.Equal(t, 0, mustHaveAny)
	assert.Equal(t, 0, mustNot)

	exact, mustHaveAny, mustNot = urn3.SpecificityTuple()
	assert.Equal(t, 1, exact)
	assert.Equal(t, 1, mustHaveAny)
	assert.Equal(t, 0, mustNot)

	exact, mustHaveAny, mustNot = urn5.SpecificityTuple()
	assert.Equal(t, 0, exact)
	assert.Equal(t, 0, mustHaveAny)
	assert.Equal(t, 1, mustNot)

	moreSpecific, err := urn2.IsMoreSpecificThan(urn1)
	require.NoError(t, err)
	assert.True(t, moreSpecific) // 3 > 2
}

func TestCompatibility(t *testing.T) {
	urn1, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf")
	require.NoError(t, err)

	urn2, err := NewTaggedUrnFromString("cap:op=generate;format=*")
	require.NoError(t, err)

	urn3, err := NewTaggedUrnFromString("cap:op=extract;ext=pdf")
	require.NoError(t, err)

	compatible, err := urn1.IsCompatibleWith(urn2)
	require.NoError(t, err)
	assert.True(t, compatible)

	compatible, err = urn2.IsCompatibleWith(urn1)
	require.NoError(t, err)
	assert.True(t, compatible)

	compatible, err = urn1.IsCompatibleWith(urn3)
	require.NoError(t, err)
	assert.False(t, compatible)

	// Missing tags are treated as wildcards for compatibility
	urn4, err := NewTaggedUrnFromString("cap:op=generate")
	require.NoError(t, err)

	compatible, err = urn1.IsCompatibleWith(urn4)
	require.NoError(t, err)
	assert.True(t, compatible)

	compatible, err = urn4.IsCompatibleWith(urn1)
	require.NoError(t, err)
	assert.True(t, compatible)
}

func TestConvenienceMethods(t *testing.T) {
	urn, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf;output=binary;target=thumbnail")
	require.NoError(t, err)

	op, exists := urn.GetTag("op")
	assert.True(t, exists)
	assert.Equal(t, "generate", op)

	target, exists := urn.GetTag("target")
	assert.True(t, exists)
	assert.Equal(t, "thumbnail", target)

	format, exists := urn.GetTag("ext")
	assert.True(t, exists)
	assert.Equal(t, "pdf", format)

	output, exists := urn.GetTag("output")
	assert.True(t, exists)
	assert.Equal(t, "binary", output)
}

func TestBuilder(t *testing.T) {
	urn, err := NewTaggedUrnBuilder("cap").
		Tag("op", "generate").
		Tag("target", "thumbnail").
		Tag("ext", "pdf").
		Tag("output", "binary").
		Build()
	require.NoError(t, err)

	op, exists := urn.GetTag("op")
	assert.True(t, exists)
	assert.Equal(t, "generate", op)

	output, exists := urn.GetTag("output")
	assert.True(t, exists)
	assert.Equal(t, "binary", output)
}

func TestWithTag(t *testing.T) {
	original, err := NewTaggedUrnFromString("cap:op=generate")
	require.NoError(t, err)

	modified := original.WithTag("ext", "pdf")

	assert.Equal(t, "cap:ext=pdf;op=generate", modified.ToString())

	// Original should be unchanged
	assert.Equal(t, "cap:op=generate", original.ToString())
}

func TestWithoutTag(t *testing.T) {
	original, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf")
	require.NoError(t, err)

	modified := original.WithoutTag("ext")

	assert.Equal(t, "cap:op=generate", modified.ToString())

	// Original should be unchanged
	assert.Equal(t, "cap:ext=pdf;op=generate", original.ToString())
}

func TestWildcardTag(t *testing.T) {
	urn, err := NewTaggedUrnFromString("cap:ext=pdf")
	require.NoError(t, err)

	wildcarded := urn.WithWildcardTag("ext")

	// Wildcard serializes as value-less tag
	assert.Equal(t, "cap:ext", wildcarded.ToString())

	// Test that wildcarded URN can match more requests
	request, err := NewTaggedUrnFromString("cap:ext=jpg")
	require.NoError(t, err)
	matches, err := urn.Matches(request)
	require.NoError(t, err)
	assert.False(t, matches)

	wildcardRequest, err := NewTaggedUrnFromString("cap:ext")
	require.NoError(t, err)
	matches, err = wildcarded.Matches(wildcardRequest)
	require.NoError(t, err)
	assert.True(t, matches)
}

func TestSubset(t *testing.T) {
	urn, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf;output=binary;target=thumbnail;")
	require.NoError(t, err)

	subset := urn.Subset([]string{"type", "ext"})

	assert.Equal(t, "cap:ext=pdf", subset.ToString())
}

func TestMerge(t *testing.T) {
	urn1, err := NewTaggedUrnFromString("cap:op=generate")
	require.NoError(t, err)

	urn2, err := NewTaggedUrnFromString("cap:ext=pdf;output=binary")
	require.NoError(t, err)

	merged, err := urn1.Merge(urn2)
	require.NoError(t, err)

	assert.Equal(t, "cap:ext=pdf;op=generate;output=binary", merged.ToString())
}

func TestMergePrefixMismatch(t *testing.T) {
	urn1, err := NewTaggedUrnFromString("cap:op=generate")
	require.NoError(t, err)

	urn2, err := NewTaggedUrnFromString("myapp:ext=pdf")
	require.NoError(t, err)

	_, err = urn1.Merge(urn2)
	assert.Error(t, err)
	capError, ok := err.(*TaggedUrnError)
	assert.True(t, ok)
	assert.Equal(t, ErrorPrefixMismatch, capError.Code)
}

func TestEquality(t *testing.T) {
	urn1, err := NewTaggedUrnFromString("cap:op=generate")
	require.NoError(t, err)

	urn2, err := NewTaggedUrnFromString("cap:op=generate") // different order
	require.NoError(t, err)

	urn3, err := NewTaggedUrnFromString("cap:op=generate;image")
	require.NoError(t, err)

	assert.True(t, urn1.Equals(urn2)) // order doesn't matter
	assert.False(t, urn1.Equals(urn3))
}

func TestEqualityDifferentPrefix(t *testing.T) {
	urn1, err := NewTaggedUrnFromString("cap:op=generate")
	require.NoError(t, err)

	urn2, err := NewTaggedUrnFromString("myapp:op=generate")
	require.NoError(t, err)

	assert.False(t, urn1.Equals(urn2))
}

func TestUrnMatcher(t *testing.T) {
	matcher := &UrnMatcher{}

	urns := []*TaggedUrn{}

	urn1, err := NewTaggedUrnFromString("cap:op=*")
	require.NoError(t, err)
	urns = append(urns, urn1)

	urn2, err := NewTaggedUrnFromString("cap:op=generate")
	require.NoError(t, err)
	urns = append(urns, urn2)

	urn3, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf")
	require.NoError(t, err)
	urns = append(urns, urn3)

	request, err := NewTaggedUrnFromString("cap:op=generate")
	require.NoError(t, err)

	best, err := matcher.FindBestMatch(urns, request)
	require.NoError(t, err)
	require.NotNil(t, best)

	// Most specific URN that can handle the request
	assert.Equal(t, "cap:ext=pdf;op=generate", best.ToString())
}

func TestUrnMatcherPrefixMismatch(t *testing.T) {
	matcher := &UrnMatcher{}

	urns := []*TaggedUrn{}

	urn1, err := NewTaggedUrnFromString("cap:op=generate")
	require.NoError(t, err)
	urns = append(urns, urn1)

	request, err := NewTaggedUrnFromString("myapp:op=generate")
	require.NoError(t, err)

	_, err = matcher.FindBestMatch(urns, request)
	assert.Error(t, err)
}

func TestJSONSerialization(t *testing.T) {
	original, err := NewTaggedUrnFromString("cap:op=generate")
	require.NoError(t, err)

	data, err := json.Marshal(original)
	assert.NoError(t, err)
	assert.NotNil(t, data)

	var decoded TaggedUrn
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.True(t, original.Equals(&decoded))
}

func TestJSONSerializationWithCustomPrefix(t *testing.T) {
	original, err := NewTaggedUrnFromString("myapp:key=value")
	require.NoError(t, err)

	data, err := json.Marshal(original)
	assert.NoError(t, err)

	var decoded TaggedUrn
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.True(t, original.Equals(&decoded))
	assert.Equal(t, "myapp", decoded.GetPrefix())
}

func TestUnquotedValuesLowercased(t *testing.T) {
	// Unquoted values are normalized to lowercase
	urn, err := NewTaggedUrnFromString("cap:OP=Generate;EXT=PDF;Target=Thumbnail;")
	require.NoError(t, err)

	// Keys are always lowercase
	op, exists := urn.GetTag("op")
	assert.True(t, exists)
	assert.Equal(t, "generate", op)

	ext, exists := urn.GetTag("ext")
	assert.True(t, exists)
	assert.Equal(t, "pdf", ext)

	target, exists := urn.GetTag("target")
	assert.True(t, exists)
	assert.Equal(t, "thumbnail", target)

	// Key lookup is case-insensitive
	op2, exists := urn.GetTag("OP")
	assert.True(t, exists)
	assert.Equal(t, "generate", op2)

	// Both URNs parse to same lowercase values
	urn2, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf;target=thumbnail;")
	require.NoError(t, err)
	assert.Equal(t, urn.ToString(), urn2.ToString())
	assert.True(t, urn.Equals(urn2))
}

func TestQuotedValuesPreserveCase(t *testing.T) {
	// Quoted values preserve their case
	urn, err := NewTaggedUrnFromString(`cap:key="Value With Spaces"`)
	require.NoError(t, err)
	value, exists := urn.GetTag("key")
	assert.True(t, exists)
	assert.Equal(t, "Value With Spaces", value)

	// Key is still lowercase
	urn2, err := NewTaggedUrnFromString(`cap:KEY="Value With Spaces"`)
	require.NoError(t, err)
	value2, exists := urn2.GetTag("key")
	assert.True(t, exists)
	assert.Equal(t, "Value With Spaces", value2)

	// Unquoted vs quoted case difference
	unquoted, err := NewTaggedUrnFromString("cap:key=UPPERCASE")
	require.NoError(t, err)
	quoted, err := NewTaggedUrnFromString(`cap:key="UPPERCASE"`)
	require.NoError(t, err)

	unquotedVal, _ := unquoted.GetTag("key")
	quotedVal, _ := quoted.GetTag("key")
	assert.Equal(t, "uppercase", unquotedVal) // lowercase
	assert.Equal(t, "UPPERCASE", quotedVal)   // preserved
	assert.False(t, unquoted.Equals(quoted))  // NOT equal
}

func TestQuotedValueSpecialChars(t *testing.T) {
	// Semicolons in quoted values
	urn, err := NewTaggedUrnFromString(`cap:key="value;with;semicolons"`)
	require.NoError(t, err)
	value, exists := urn.GetTag("key")
	assert.True(t, exists)
	assert.Equal(t, "value;with;semicolons", value)

	// Equals in quoted values
	urn2, err := NewTaggedUrnFromString(`cap:key="value=with=equals"`)
	require.NoError(t, err)
	value2, exists := urn2.GetTag("key")
	assert.True(t, exists)
	assert.Equal(t, "value=with=equals", value2)

	// Spaces in quoted values
	urn3, err := NewTaggedUrnFromString(`cap:key="hello world"`)
	require.NoError(t, err)
	value3, exists := urn3.GetTag("key")
	assert.True(t, exists)
	assert.Equal(t, "hello world", value3)
}

func TestQuotedValueEscapeSequences(t *testing.T) {
	// Escaped quotes
	urn, err := NewTaggedUrnFromString(`cap:key="value\"quoted\""`)
	require.NoError(t, err)
	value, exists := urn.GetTag("key")
	assert.True(t, exists)
	assert.Equal(t, `value"quoted"`, value)

	// Escaped backslashes
	urn2, err := NewTaggedUrnFromString(`cap:key="path\\file"`)
	require.NoError(t, err)
	value2, exists := urn2.GetTag("key")
	assert.True(t, exists)
	assert.Equal(t, `path\file`, value2)

	// Mixed escapes
	urn3, err := NewTaggedUrnFromString(`cap:key="say \"hello\\world\""`)
	require.NoError(t, err)
	value3, exists := urn3.GetTag("key")
	assert.True(t, exists)
	assert.Equal(t, `say "hello\world"`, value3)
}

func TestMixedQuotedUnquoted(t *testing.T) {
	urn, err := NewTaggedUrnFromString(`cap:a="Quoted";b=simple`)
	require.NoError(t, err)

	a, exists := urn.GetTag("a")
	assert.True(t, exists)
	assert.Equal(t, "Quoted", a)

	b, exists := urn.GetTag("b")
	assert.True(t, exists)
	assert.Equal(t, "simple", b)
}

func TestUnterminatedQuoteError(t *testing.T) {
	urn, err := NewTaggedUrnFromString(`cap:key="unterminated`)
	assert.Nil(t, urn)
	assert.Error(t, err)
	urnError, ok := err.(*TaggedUrnError)
	assert.True(t, ok)
	assert.Equal(t, ErrorUnterminatedQuote, urnError.Code)
}

func TestInvalidEscapeSequenceError(t *testing.T) {
	urn, err := NewTaggedUrnFromString(`cap:key="bad\n"`)
	assert.Nil(t, urn)
	assert.Error(t, err)
	urnError, ok := err.(*TaggedUrnError)
	assert.True(t, ok)
	assert.Equal(t, ErrorInvalidEscapeSequence, urnError.Code)

	// Invalid escape at end
	urn2, err := NewTaggedUrnFromString(`cap:key="bad\x"`)
	assert.Nil(t, urn2)
	assert.Error(t, err)
	urnError2, ok := err.(*TaggedUrnError)
	assert.True(t, ok)
	assert.Equal(t, ErrorInvalidEscapeSequence, urnError2.Code)
}

func TestSerializationSmartQuoting(t *testing.T) {
	// Simple lowercase value - no quoting needed
	urn, err := NewTaggedUrnBuilder("cap").Tag("key", "simple").Build()
	require.NoError(t, err)
	assert.Equal(t, "cap:key=simple", urn.ToString())

	// Value with spaces - needs quoting
	urn2, err := NewTaggedUrnBuilder("cap").Tag("key", "has spaces").Build()
	require.NoError(t, err)
	assert.Equal(t, `cap:key="has spaces"`, urn2.ToString())

	// Value with semicolons - needs quoting
	urn3, err := NewTaggedUrnBuilder("cap").Tag("key", "has;semi").Build()
	require.NoError(t, err)
	assert.Equal(t, `cap:key="has;semi"`, urn3.ToString())

	// Value with uppercase - needs quoting to preserve
	urn4, err := NewTaggedUrnBuilder("cap").Tag("key", "HasUpper").Build()
	require.NoError(t, err)
	assert.Equal(t, `cap:key="HasUpper"`, urn4.ToString())

	// Value with quotes - needs quoting and escaping
	urn5, err := NewTaggedUrnBuilder("cap").Tag("key", `has"quote`).Build()
	require.NoError(t, err)
	assert.Equal(t, `cap:key="has\"quote"`, urn5.ToString())

	// Value with backslashes - needs quoting and escaping
	urn6, err := NewTaggedUrnBuilder("cap").Tag("key", `path\file`).Build()
	require.NoError(t, err)
	assert.Equal(t, `cap:key="path\\file"`, urn6.ToString())
}

func TestRoundTripSimple(t *testing.T) {
	original := "cap:op=generate;ext=pdf"
	urn, err := NewTaggedUrnFromString(original)
	require.NoError(t, err)
	serialized := urn.ToString()
	reparsed, err := NewTaggedUrnFromString(serialized)
	require.NoError(t, err)
	assert.True(t, urn.Equals(reparsed))
}

func TestRoundTripQuoted(t *testing.T) {
	original := `cap:key="Value With Spaces"`
	urn, err := NewTaggedUrnFromString(original)
	require.NoError(t, err)
	serialized := urn.ToString()
	reparsed, err := NewTaggedUrnFromString(serialized)
	require.NoError(t, err)
	assert.True(t, urn.Equals(reparsed))
	value, exists := reparsed.GetTag("key")
	assert.True(t, exists)
	assert.Equal(t, "Value With Spaces", value)
}

func TestRoundTripEscapes(t *testing.T) {
	original := `cap:key="value\"with\\escapes"`
	urn, err := NewTaggedUrnFromString(original)
	require.NoError(t, err)
	value, exists := urn.GetTag("key")
	assert.True(t, exists)
	assert.Equal(t, `value"with\escapes`, value)
	serialized := urn.ToString()
	reparsed, err := NewTaggedUrnFromString(serialized)
	require.NoError(t, err)
	assert.True(t, urn.Equals(reparsed))
}

func TestMatchingCaseSensitiveValues(t *testing.T) {
	// Values with different case should NOT match
	urn1, err := NewTaggedUrnFromString(`cap:key="Value"`)
	require.NoError(t, err)
	urn2, err := NewTaggedUrnFromString(`cap:key="value"`)
	require.NoError(t, err)

	matches1, err := urn1.Matches(urn2)
	require.NoError(t, err)
	assert.False(t, matches1)

	matches2, err := urn2.Matches(urn1)
	require.NoError(t, err)
	assert.False(t, matches2)

	// Same case should match
	urn3, err := NewTaggedUrnFromString(`cap:key="Value"`)
	require.NoError(t, err)
	matches3, err := urn1.Matches(urn3)
	require.NoError(t, err)
	assert.True(t, matches3)
}

func TestBuilderPreservesCase(t *testing.T) {
	urn, err := NewTaggedUrnBuilder("cap").
		Tag("KEY", "ValueWithCase").
		Build()
	require.NoError(t, err)

	// Key is lowercase
	value, exists := urn.GetTag("key")
	assert.True(t, exists)
	assert.Equal(t, "ValueWithCase", value)

	// Value case preserved, so needs quoting
	assert.Equal(t, `cap:key="ValueWithCase"`, urn.ToString())
}

func TestHasTagCaseSensitive(t *testing.T) {
	urn, err := NewTaggedUrnFromString(`cap:key="Value"`)
	require.NoError(t, err)

	// Exact case match works
	assert.True(t, urn.HasTag("key", "Value"))

	// Different case does not match
	assert.False(t, urn.HasTag("key", "value"))
	assert.False(t, urn.HasTag("key", "VALUE"))

	// Key lookup is case-insensitive
	assert.True(t, urn.HasTag("KEY", "Value"))
	assert.True(t, urn.HasTag("Key", "Value"))
}

func TestWithTagPreservesValue(t *testing.T) {
	urn := NewTaggedUrnFromTags("cap", map[string]string{})
	modified := urn.WithTag("key", "ValueWithCase")

	value, exists := modified.GetTag("key")
	assert.True(t, exists)
	assert.Equal(t, "ValueWithCase", value)
}

func TestSemanticEquivalence(t *testing.T) {
	// Unquoted and quoted simple lowercase values are equivalent
	unquoted, err := NewTaggedUrnFromString("cap:key=simple")
	require.NoError(t, err)
	quoted, err := NewTaggedUrnFromString(`cap:key="simple"`)
	require.NoError(t, err)
	assert.True(t, unquoted.Equals(quoted))

	// Both serialize the same way (unquoted)
	assert.Equal(t, "cap:key=simple", unquoted.ToString())
	assert.Equal(t, "cap:key=simple", quoted.ToString())
}

func TestEmptyTaggedUrn(t *testing.T) {
	// Empty tagged URN is valid
	empty, err := NewTaggedUrnFromString("cap:")
	assert.NoError(t, err)
	assert.NotNil(t, empty)
	assert.Equal(t, "cap:", empty.ToString())

	// NEW SEMANTICS:
	// Empty PATTERN matches any INSTANCE (pattern has no constraints)
	// Empty INSTANCE only matches patterns that have no required tags

	specific, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf")
	assert.NoError(t, err)

	// Empty instance vs specific pattern: NO MATCH
	// Pattern requires op=generate and ext=pdf, instance doesn't have them
	matches, err := empty.Matches(specific)
	require.NoError(t, err)
	assert.False(t, matches)

	// Specific instance vs empty pattern: MATCH
	// Pattern has no constraints, instance can have anything
	matches, err = specific.Matches(empty)
	require.NoError(t, err)
	assert.True(t, matches)

	// Empty instance vs empty pattern: MATCH
	matches, err = empty.Matches(empty)
	require.NoError(t, err)
	assert.True(t, matches)

	// With trailing semicolon
	empty2, err := NewTaggedUrnFromString("cap:;")
	assert.NoError(t, err)
	assert.Equal(t, "cap:", empty2.ToString())
}

func TestEmptyWithCustomPrefix(t *testing.T) {
	empty, err := NewTaggedUrnFromString("myapp:")
	require.NoError(t, err)
	assert.Equal(t, "myapp", empty.GetPrefix())
	assert.Equal(t, "myapp:", empty.ToString())
}

func TestExtendedCharacterSupport(t *testing.T) {
	// Test forward slashes and colons in tag components
	urn, err := NewTaggedUrnFromString("cap:url=https://example_org/api;path=/some/file")
	assert.NoError(t, err)
	assert.NotNil(t, urn)

	url, exists := urn.GetTag("url")
	assert.True(t, exists)
	assert.Equal(t, "https://example_org/api", url)

	path, exists := urn.GetTag("path")
	assert.True(t, exists)
	assert.Equal(t, "/some/file", path)
}

func TestWildcardRestrictions(t *testing.T) {
	// Wildcard should be rejected in keys
	invalidKey, err := NewTaggedUrnFromString("cap:*=value")
	assert.Error(t, err)
	assert.Nil(t, invalidKey)
	urnError, ok := err.(*TaggedUrnError)
	assert.True(t, ok)
	assert.Equal(t, ErrorInvalidCharacter, urnError.Code)

	// Wildcard should be accepted in values
	validValue, err := NewTaggedUrnFromString("cap:key=*")
	assert.NoError(t, err)
	assert.NotNil(t, validValue)

	value, exists := validValue.GetTag("key")
	assert.True(t, exists)
	assert.Equal(t, "*", value)
}

func TestDuplicateKeyRejection(t *testing.T) {
	// Duplicate keys should be rejected
	duplicate, err := NewTaggedUrnFromString("cap:key=value1;key=value2")
	assert.Error(t, err)
	assert.Nil(t, duplicate)
	urnError, ok := err.(*TaggedUrnError)
	assert.True(t, ok)
	assert.Equal(t, ErrorDuplicateKey, urnError.Code)
}

func TestNumericKeyRestriction(t *testing.T) {
	// Pure numeric keys should be rejected
	numericKey, err := NewTaggedUrnFromString("cap:123=value")
	assert.Error(t, err)
	assert.Nil(t, numericKey)
	urnError, ok := err.(*TaggedUrnError)
	assert.True(t, ok)
	assert.Equal(t, ErrorNumericKey, urnError.Code)

	// Mixed alphanumeric keys should be allowed
	mixedKey1, err := NewTaggedUrnFromString("cap:key123=value")
	assert.NoError(t, err)
	assert.NotNil(t, mixedKey1)

	mixedKey2, err := NewTaggedUrnFromString("cap:123key=value")
	assert.NoError(t, err)
	assert.NotNil(t, mixedKey2)

	// Pure numeric values should be allowed
	numericValue, err := NewTaggedUrnFromString("cap:key=123")
	assert.NoError(t, err)
	assert.NotNil(t, numericValue)

	value, exists := numericValue.GetTag("key")
	assert.True(t, exists)
	assert.Equal(t, "123", value)
}

func TestEmptyValueError(t *testing.T) {
	urn, err := NewTaggedUrnFromString("cap:key=")
	assert.Nil(t, urn)
	assert.Error(t, err)

	urn2, err := NewTaggedUrnFromString("cap:key=;other=value")
	assert.Nil(t, urn2)
	assert.Error(t, err)
}

func TestMatchingDifferentPrefixesError(t *testing.T) {
	// URNs with different prefixes should cause an error, not just return false
	urn1, err := NewTaggedUrnFromString("cap:op=test")
	require.NoError(t, err)
	urn2, err := NewTaggedUrnFromString("other:op=test")
	require.NoError(t, err)

	_, err = urn1.Matches(urn2)
	assert.Error(t, err)

	_, err = urn1.IsCompatibleWith(urn2)
	assert.Error(t, err)

	_, err = urn1.IsMoreSpecificThan(urn2)
	assert.Error(t, err)
}

// ============================================================================
// MATCHING SEMANTICS SPECIFICATION TESTS
// These 9 tests verify the exact matching semantics from RULES.md Sections 12-17
// All implementations (Rust, Go, JS, ObjC) must pass these identically
// ============================================================================

func TestMatchingSemantics_Test1_ExactMatch(t *testing.T) {
	// Test 1: Exact match
	// URN:     cap:op=generate;ext=pdf
	// Request: cap:op=generate;ext=pdf
	// Result:  MATCH
	urn, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf")
	require.NoError(t, err)

	request, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf")
	require.NoError(t, err)

	matches, err := urn.Matches(request)
	require.NoError(t, err)
	assert.True(t, matches, "Test 1: Exact match should succeed")
}

func TestMatchingSemantics_Test2_InstanceMissingTag(t *testing.T) {
	// Test 2: Instance missing tag
	// Instance: cap:op=generate
	// Pattern:  cap:op=generate;ext=pdf
	// Result:   NO MATCH (pattern requires ext=pdf, instance doesn't have ext)
	//
	// NEW SEMANTICS: Missing tag in instance means it doesn't exist.
	// Pattern K=v requires instance to have K=v.
	instance, err := NewTaggedUrnFromString("cap:op=generate")
	require.NoError(t, err)

	pattern, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf")
	require.NoError(t, err)

	matches, err := instance.Matches(pattern)
	require.NoError(t, err)
	assert.False(t, matches, "Test 2: Instance missing tag should NOT match when pattern requires it")

	// To accept any ext (or missing), use pattern with ext=?
	patternOptional, err := NewTaggedUrnFromString("cap:op=generate;ext=?")
	require.NoError(t, err)
	matches, err = instance.Matches(patternOptional)
	require.NoError(t, err)
	assert.True(t, matches, "Pattern with ext=? should match instance without ext")
}

func TestMatchingSemantics_Test3_UrnHasExtraTag(t *testing.T) {
	// Test 3: URN has extra tag
	// URN:     cap:op=generate;ext=pdf;version=2
	// Request: cap:op=generate;ext=pdf
	// Result:  MATCH (request doesn't constrain version)
	urn, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf;version=2")
	require.NoError(t, err)

	request, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf")
	require.NoError(t, err)

	matches, err := urn.Matches(request)
	require.NoError(t, err)
	assert.True(t, matches, "Test 3: URN with extra tag should match")
}

func TestMatchingSemantics_Test4_RequestHasWildcard(t *testing.T) {
	// Test 4: Request has wildcard
	// URN:     cap:op=generate;ext=pdf
	// Request: cap:op=generate;ext=*
	// Result:  MATCH (request accepts any ext)
	urn, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf")
	require.NoError(t, err)

	request, err := NewTaggedUrnFromString("cap:op=generate;ext=*")
	require.NoError(t, err)

	matches, err := urn.Matches(request)
	require.NoError(t, err)
	assert.True(t, matches, "Test 4: Request wildcard should match")
}

func TestMatchingSemantics_Test5_UrnHasWildcard(t *testing.T) {
	// Test 5: URN has wildcard
	// URN:     cap:op=generate;ext=*
	// Request: cap:op=generate;ext=pdf
	// Result:  MATCH (URN handles any ext)
	urn, err := NewTaggedUrnFromString("cap:op=generate;ext=*")
	require.NoError(t, err)

	request, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf")
	require.NoError(t, err)

	matches, err := urn.Matches(request)
	require.NoError(t, err)
	assert.True(t, matches, "Test 5: URN wildcard should match")
}

func TestMatchingSemantics_Test6_ValueMismatch(t *testing.T) {
	// Test 6: Value mismatch
	// URN:     cap:op=generate;ext=pdf
	// Request: cap:op=generate;ext=docx
	// Result:  NO MATCH
	urn, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf")
	require.NoError(t, err)

	request, err := NewTaggedUrnFromString("cap:op=generate;ext=docx")
	require.NoError(t, err)

	matches, err := urn.Matches(request)
	require.NoError(t, err)
	assert.False(t, matches, "Test 6: Value mismatch should not match")
}

func TestMatchingSemantics_Test7_PatternHasExtraTag(t *testing.T) {
	// Test 7: Pattern has extra tag that instance doesn't have
	// Instance: cap:op=generate_thumbnail;out="media:binary"
	// Pattern:  cap:op=generate_thumbnail;out="media:binary";ext=wav
	// Result:   NO MATCH (pattern requires ext=wav, instance doesn't have ext)
	//
	// NEW SEMANTICS: Pattern K=v requires instance to have K=v
	instance, err := NewTaggedUrnFromString(`cap:op=generate_thumbnail;out="media:binary"`)
	require.NoError(t, err)

	pattern, err := NewTaggedUrnFromString(`cap:op=generate_thumbnail;out="media:binary";ext=wav`)
	require.NoError(t, err)

	matches, err := instance.Matches(pattern)
	require.NoError(t, err)
	assert.False(t, matches, "Test 7: Instance missing ext should NOT match when pattern requires ext=wav")

	// Instance vs pattern that doesn't constrain ext: MATCH
	patternNoExt, err := NewTaggedUrnFromString(`cap:op=generate_thumbnail;out="media:binary"`)
	require.NoError(t, err)
	matches, err = instance.Matches(patternNoExt)
	require.NoError(t, err)
	assert.True(t, matches)
}

func TestMatchingSemantics_Test8_EmptyPatternMatchesAnything(t *testing.T) {
	// Test 8: Empty PATTERN matches any INSTANCE
	// Instance: cap:op=generate;ext=pdf
	// Pattern:  cap:
	// Result:   MATCH (pattern has no constraints)
	//
	// NEW SEMANTICS: Empty pattern = no constraints = matches any instance
	// But empty instance only matches patterns that don't require tags
	instance, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf")
	require.NoError(t, err)

	emptyPattern, err := NewTaggedUrnFromString("cap:")
	require.NoError(t, err)

	matches, err := instance.Matches(emptyPattern)
	require.NoError(t, err)
	assert.True(t, matches, "Test 8: Any instance should match empty pattern")

	// Empty instance vs pattern with requirements: NO MATCH
	emptyInstance, err := NewTaggedUrnFromString("cap:")
	require.NoError(t, err)
	pattern, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf")
	require.NoError(t, err)
	matches, err = emptyInstance.Matches(pattern)
	require.NoError(t, err)
	assert.False(t, matches, "Empty instance should NOT match pattern with requirements")
}

func TestMatchingSemantics_Test9_CrossDimensionConstraints(t *testing.T) {
	// Test 9: Cross-dimension constraints
	// Instance: cap:op=generate
	// Pattern:  cap:ext=pdf
	// Result:   NO MATCH (pattern requires ext=pdf, instance doesn't have ext)
	//
	// NEW SEMANTICS: Pattern K=v requires instance to have K=v
	instance, err := NewTaggedUrnFromString("cap:op=generate")
	require.NoError(t, err)

	pattern, err := NewTaggedUrnFromString("cap:ext=pdf")
	require.NoError(t, err)

	matches, err := instance.Matches(pattern)
	require.NoError(t, err)
	assert.False(t, matches, "Test 9: Instance without ext should NOT match pattern requiring ext")

	// Instance with ext vs pattern with different tag only: MATCH
	instance2, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf")
	require.NoError(t, err)
	pattern2, err := NewTaggedUrnFromString("cap:ext=pdf")
	require.NoError(t, err)
	matches, err = instance2.Matches(pattern2)
	require.NoError(t, err)
	assert.True(t, matches, "Instance with ext=pdf should match pattern requiring ext=pdf")
}

// ============================================================================
// VALUE-LESS TAG TESTS
// Value-less tags are equivalent to wildcard tags (key=*)
// ============================================================================

func TestValuelessTagParsingSingle(t *testing.T) {
	// Single value-less tag
	urn, err := NewTaggedUrnFromString("cap:optimize")
	require.NoError(t, err)

	value, exists := urn.GetTag("optimize")
	assert.True(t, exists)
	assert.Equal(t, "*", value)
	// Serializes as value-less (no =*)
	assert.Equal(t, "cap:optimize", urn.ToString())
}

func TestValuelessTagParsingMultiple(t *testing.T) {
	// Multiple value-less tags
	urn, err := NewTaggedUrnFromString("cap:fast;optimize;secure")
	require.NoError(t, err)

	fast, exists := urn.GetTag("fast")
	assert.True(t, exists)
	assert.Equal(t, "*", fast)

	optimize, exists := urn.GetTag("optimize")
	assert.True(t, exists)
	assert.Equal(t, "*", optimize)

	secure, exists := urn.GetTag("secure")
	assert.True(t, exists)
	assert.Equal(t, "*", secure)

	// Serializes alphabetically as value-less
	assert.Equal(t, "cap:fast;optimize;secure", urn.ToString())
}

func TestValuelessTagMixedWithValued(t *testing.T) {
	// Mix of value-less and valued tags
	urn, err := NewTaggedUrnFromString("cap:op=generate;optimize;ext=pdf;secure")
	require.NoError(t, err)

	op, exists := urn.GetTag("op")
	assert.True(t, exists)
	assert.Equal(t, "generate", op)

	optimize, exists := urn.GetTag("optimize")
	assert.True(t, exists)
	assert.Equal(t, "*", optimize)

	ext, exists := urn.GetTag("ext")
	assert.True(t, exists)
	assert.Equal(t, "pdf", ext)

	secure, exists := urn.GetTag("secure")
	assert.True(t, exists)
	assert.Equal(t, "*", secure)

	// Serializes alphabetically
	assert.Equal(t, "cap:ext=pdf;op=generate;optimize;secure", urn.ToString())
}

func TestValuelessTagAtEnd(t *testing.T) {
	// Value-less tag at the end (no trailing semicolon)
	urn, err := NewTaggedUrnFromString("cap:op=generate;optimize")
	require.NoError(t, err)

	op, exists := urn.GetTag("op")
	assert.True(t, exists)
	assert.Equal(t, "generate", op)

	optimize, exists := urn.GetTag("optimize")
	assert.True(t, exists)
	assert.Equal(t, "*", optimize)

	assert.Equal(t, "cap:op=generate;optimize", urn.ToString())
}

func TestValuelessTagEquivalenceToWildcard(t *testing.T) {
	// Value-less tag is equivalent to explicit wildcard
	valueless, err := NewTaggedUrnFromString("cap:ext")
	require.NoError(t, err)

	wildcard, err := NewTaggedUrnFromString("cap:ext=*")
	require.NoError(t, err)

	assert.True(t, valueless.Equals(wildcard))
	// Both serialize to value-less form
	assert.Equal(t, "cap:ext", valueless.ToString())
	assert.Equal(t, "cap:ext", wildcard.ToString())
}

func TestValuelessTagMatching(t *testing.T) {
	// Value-less tag (wildcard) matches any value
	urn, err := NewTaggedUrnFromString("cap:op=generate;ext")
	require.NoError(t, err)

	requestPdf, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf")
	require.NoError(t, err)
	requestDocx, err := NewTaggedUrnFromString("cap:op=generate;ext=docx")
	require.NoError(t, err)
	requestAny, err := NewTaggedUrnFromString("cap:op=generate;ext=anything")
	require.NoError(t, err)

	matches, err := urn.Matches(requestPdf)
	require.NoError(t, err)
	assert.True(t, matches)

	matches, err = urn.Matches(requestDocx)
	require.NoError(t, err)
	assert.True(t, matches)

	matches, err = urn.Matches(requestAny)
	require.NoError(t, err)
	assert.True(t, matches)
}

func TestValuelessTagInPattern(t *testing.T) {
	// Pattern with value-less tag (K=*) requires instance to have the tag
	pattern, err := NewTaggedUrnFromString("cap:op=generate;ext")
	require.NoError(t, err)

	instancePdf, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf")
	require.NoError(t, err)
	instanceDocx, err := NewTaggedUrnFromString("cap:op=generate;ext=docx")
	require.NoError(t, err)
	instanceMissing, err := NewTaggedUrnFromString("cap:op=generate")
	require.NoError(t, err)

	// NEW SEMANTICS: K=* (valueless tag) means must-have-any
	matches, err := instancePdf.Matches(pattern)
	require.NoError(t, err)
	assert.True(t, matches) // Has ext=pdf

	matches, err = instanceDocx.Matches(pattern)
	require.NoError(t, err)
	assert.True(t, matches) // Has ext=docx

	matches, err = instanceMissing.Matches(pattern)
	require.NoError(t, err)
	assert.False(t, matches) // Missing ext, pattern requires it

	// To accept missing ext, use ? instead
	patternOptional, err := NewTaggedUrnFromString("cap:op=generate;ext=?")
	require.NoError(t, err)
	matches, err = instanceMissing.Matches(patternOptional)
	require.NoError(t, err)
	assert.True(t, matches)
}

func TestValuelessTagSpecificity(t *testing.T) {
	// NEW GRADED SPECIFICITY:
	// K=v (exact): 3, K=* (must-have-any): 2, K=! (must-not): 1, K=? (unspecified): 0

	urn1, err := NewTaggedUrnFromString("cap:op=generate")
	require.NoError(t, err)
	urn2, err := NewTaggedUrnFromString("cap:op=generate;optimize") // optimize = *
	require.NoError(t, err)
	urn3, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf")
	require.NoError(t, err)

	assert.Equal(t, 3, urn1.Specificity())  // 1 exact = 3
	assert.Equal(t, 5, urn2.Specificity())  // 1 exact + 1 * = 3 + 2 = 5
	assert.Equal(t, 6, urn3.Specificity())  // 2 exact = 3 + 3 = 6
}

func TestValuelessTagRoundtrip(t *testing.T) {
	// Round-trip parsing and serialization
	original := "cap:ext=pdf;op=generate;optimize;secure"
	urn, err := NewTaggedUrnFromString(original)
	require.NoError(t, err)
	serialized := urn.ToString()
	reparsed, err := NewTaggedUrnFromString(serialized)
	require.NoError(t, err)
	assert.True(t, urn.Equals(reparsed))
	assert.Equal(t, original, serialized)
}

func TestValuelessTagCaseNormalization(t *testing.T) {
	// Value-less tags are normalized to lowercase like other keys
	urn, err := NewTaggedUrnFromString("cap:OPTIMIZE;Fast;SECURE")
	require.NoError(t, err)

	optimize, exists := urn.GetTag("optimize")
	assert.True(t, exists)
	assert.Equal(t, "*", optimize)

	fast, exists := urn.GetTag("fast")
	assert.True(t, exists)
	assert.Equal(t, "*", fast)

	secure, exists := urn.GetTag("secure")
	assert.True(t, exists)
	assert.Equal(t, "*", secure)

	assert.Equal(t, "cap:fast;optimize;secure", urn.ToString())
}

func TestEmptyValueStillError(t *testing.T) {
	// Empty value with = is still an error (different from value-less)
	urn, err := NewTaggedUrnFromString("cap:key=")
	assert.Nil(t, urn)
	assert.Error(t, err)

	urn2, err := NewTaggedUrnFromString("cap:key=;other=value")
	assert.Nil(t, urn2)
	assert.Error(t, err)
}

func TestValuelessTagCompatibility(t *testing.T) {
	// Value-less tags are compatible with any value
	urn1, err := NewTaggedUrnFromString("cap:op=generate;ext")
	require.NoError(t, err)
	urn2, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf")
	require.NoError(t, err)
	urn3, err := NewTaggedUrnFromString("cap:op=generate;ext=docx")
	require.NoError(t, err)

	compatible, err := urn1.IsCompatibleWith(urn2)
	require.NoError(t, err)
	assert.True(t, compatible)

	compatible, err = urn1.IsCompatibleWith(urn3)
	require.NoError(t, err)
	assert.True(t, compatible)

	// But urn2 and urn3 are not compatible (different specific values)
	compatible, err = urn2.IsCompatibleWith(urn3)
	require.NoError(t, err)
	assert.False(t, compatible)
}

func TestValuelessNumericKeyStillRejected(t *testing.T) {
	// Purely numeric keys are still rejected for value-less tags
	urn, err := NewTaggedUrnFromString("cap:123")
	assert.Nil(t, urn)
	assert.Error(t, err)

	urn2, err := NewTaggedUrnFromString("cap:op=generate;456")
	assert.Nil(t, urn2)
	assert.Error(t, err)
}

func TestWhitespaceInInputRejected(t *testing.T) {
	// Leading whitespace fails hard
	urn, err := NewTaggedUrnFromString(" cap:op=test")
	assert.Nil(t, urn)
	assert.Error(t, err)
	urnError, ok := err.(*TaggedUrnError)
	assert.True(t, ok)
	assert.Equal(t, ErrorWhitespaceInInput, urnError.Code)

	// Trailing whitespace fails hard
	urn, err = NewTaggedUrnFromString("cap:op=test ")
	assert.Nil(t, urn)
	assert.Error(t, err)
	urnError, ok = err.(*TaggedUrnError)
	assert.True(t, ok)
	assert.Equal(t, ErrorWhitespaceInInput, urnError.Code)

	// Both leading and trailing whitespace fails hard
	urn, err = NewTaggedUrnFromString(" cap:op=test ")
	assert.Nil(t, urn)
	assert.Error(t, err)
	urnError, ok = err.(*TaggedUrnError)
	assert.True(t, ok)
	assert.Equal(t, ErrorWhitespaceInInput, urnError.Code)

	// Tab and newline also count as whitespace
	urn, err = NewTaggedUrnFromString("\tcap:op=test")
	assert.Nil(t, urn)
	assert.Error(t, err)

	urn, err = NewTaggedUrnFromString("cap:op=test\n")
	assert.Nil(t, urn)
	assert.Error(t, err)

	// Clean input works
	urn, err = NewTaggedUrnFromString("cap:op=test")
	assert.NoError(t, err)
	assert.NotNil(t, urn)
	value, exists := urn.GetTag("op")
	assert.True(t, exists)
	assert.Equal(t, "test", value)
}

// ============================================================================
// NEW SEMANTICS TESTS: ? (unspecified) and ! (must-not-have)
// ============================================================================

func TestUnspecifiedQuestionMarkParsing(t *testing.T) {
	// ? parses as unspecified
	urn, err := NewTaggedUrnFromString("cap:ext=?")
	require.NoError(t, err)

	value, exists := urn.GetTag("ext")
	assert.True(t, exists)
	assert.Equal(t, "?", value)
	// Serializes as key=?
	assert.Equal(t, "cap:ext=?", urn.ToString())
}

func TestMustNotHaveExclamationParsing(t *testing.T) {
	// ! parses as must-not-have
	urn, err := NewTaggedUrnFromString("cap:ext=!")
	require.NoError(t, err)

	value, exists := urn.GetTag("ext")
	assert.True(t, exists)
	assert.Equal(t, "!", value)
	// Serializes as key=!
	assert.Equal(t, "cap:ext=!", urn.ToString())
}

func TestQuestionMarkPatternMatchesAnything(t *testing.T) {
	// Pattern with K=? matches any instance (with or without K)
	pattern, err := NewTaggedUrnFromString("cap:ext=?")
	require.NoError(t, err)

	instancePdf, _ := NewTaggedUrnFromString("cap:ext=pdf")
	instanceDocx, _ := NewTaggedUrnFromString("cap:ext=docx")
	instanceMissing, _ := NewTaggedUrnFromString("cap:")
	instanceWildcard, _ := NewTaggedUrnFromString("cap:ext=*")
	instanceMustNot, _ := NewTaggedUrnFromString("cap:ext=!")

	matches, _ := instancePdf.Matches(pattern)
	assert.True(t, matches, "ext=pdf should match ext=?")

	matches, _ = instanceDocx.Matches(pattern)
	assert.True(t, matches, "ext=docx should match ext=?")

	matches, _ = instanceMissing.Matches(pattern)
	assert.True(t, matches, "(no ext) should match ext=?")

	matches, _ = instanceWildcard.Matches(pattern)
	assert.True(t, matches, "ext=* should match ext=?")

	matches, _ = instanceMustNot.Matches(pattern)
	assert.True(t, matches, "ext=! should match ext=?")
}

func TestQuestionMarkInInstance(t *testing.T) {
	// Instance with K=? matches any pattern constraint
	instance, err := NewTaggedUrnFromString("cap:ext=?")
	require.NoError(t, err)

	patternPdf, _ := NewTaggedUrnFromString("cap:ext=pdf")
	patternWildcard, _ := NewTaggedUrnFromString("cap:ext=*")
	patternMustNot, _ := NewTaggedUrnFromString("cap:ext=!")
	patternQuestion, _ := NewTaggedUrnFromString("cap:ext=?")
	patternMissing, _ := NewTaggedUrnFromString("cap:")

	matches, _ := instance.Matches(patternPdf)
	assert.True(t, matches, "ext=? should match ext=pdf")

	matches, _ = instance.Matches(patternWildcard)
	assert.True(t, matches, "ext=? should match ext=*")

	matches, _ = instance.Matches(patternMustNot)
	assert.True(t, matches, "ext=? should match ext=!")

	matches, _ = instance.Matches(patternQuestion)
	assert.True(t, matches, "ext=? should match ext=?")

	matches, _ = instance.Matches(patternMissing)
	assert.True(t, matches, "ext=? should match (no ext)")
}

func TestMustNotHavePatternRequiresAbsent(t *testing.T) {
	// Pattern with K=! requires instance to NOT have K
	pattern, err := NewTaggedUrnFromString("cap:ext=!")
	require.NoError(t, err)

	instanceMissing, _ := NewTaggedUrnFromString("cap:")
	instancePdf, _ := NewTaggedUrnFromString("cap:ext=pdf")
	instanceWildcard, _ := NewTaggedUrnFromString("cap:ext=*")
	instanceMustNot, _ := NewTaggedUrnFromString("cap:ext=!")

	matches, _ := instanceMissing.Matches(pattern)
	assert.True(t, matches, "(no ext) should match ext=!")

	matches, _ = instancePdf.Matches(pattern)
	assert.False(t, matches, "ext=pdf should NOT match ext=!")

	matches, _ = instanceWildcard.Matches(pattern)
	assert.False(t, matches, "ext=* should NOT match ext=!")

	matches, _ = instanceMustNot.Matches(pattern)
	assert.True(t, matches, "ext=! should match ext=!")
}

func TestMustNotHaveInInstance(t *testing.T) {
	// Instance with K=! conflicts with patterns requiring K
	instance, err := NewTaggedUrnFromString("cap:ext=!")
	require.NoError(t, err)

	patternPdf, _ := NewTaggedUrnFromString("cap:ext=pdf")
	patternWildcard, _ := NewTaggedUrnFromString("cap:ext=*")
	patternMustNot, _ := NewTaggedUrnFromString("cap:ext=!")
	patternQuestion, _ := NewTaggedUrnFromString("cap:ext=?")
	patternMissing, _ := NewTaggedUrnFromString("cap:")

	matches, _ := instance.Matches(patternPdf)
	assert.False(t, matches, "ext=! should NOT match ext=pdf")

	matches, _ = instance.Matches(patternWildcard)
	assert.False(t, matches, "ext=! should NOT match ext=*")

	matches, _ = instance.Matches(patternMustNot)
	assert.True(t, matches, "ext=! should match ext=!")

	matches, _ = instance.Matches(patternQuestion)
	assert.True(t, matches, "ext=! should match ext=?")

	matches, _ = instance.Matches(patternMissing)
	assert.True(t, matches, "ext=! should match (no ext)")
}

func TestFullCrossProductMatching(t *testing.T) {
	// Comprehensive test of all instance/pattern combinations
	check := func(instance, pattern string, expected bool, msg string) {
		inst, err := NewTaggedUrnFromString(instance)
		require.NoError(t, err)
		patt, err := NewTaggedUrnFromString(pattern)
		require.NoError(t, err)
		matches, err := inst.Matches(patt)
		require.NoError(t, err)
		assert.Equal(t, expected, matches, "%s: instance=%s, pattern=%s", msg, instance, pattern)
	}

	// Instance missing, Pattern variations
	check("cap:", "cap:", true, "(none)/(none)")
	check("cap:", "cap:k=?", true, "(none)/K=?")
	check("cap:", "cap:k=!", true, "(none)/K=!")
	check("cap:", "cap:k", false, "(none)/K=*")
	check("cap:", "cap:k=v", false, "(none)/K=v")

	// Instance K=?, Pattern variations
	check("cap:k=?", "cap:", true, "K=?/(none)")
	check("cap:k=?", "cap:k=?", true, "K=?/K=?")
	check("cap:k=?", "cap:k=!", true, "K=?/K=!")
	check("cap:k=?", "cap:k", true, "K=?/K=*")
	check("cap:k=?", "cap:k=v", true, "K=?/K=v")

	// Instance K=!, Pattern variations
	check("cap:k=!", "cap:", true, "K=!/(none)")
	check("cap:k=!", "cap:k=?", true, "K=!/K=?")
	check("cap:k=!", "cap:k=!", true, "K=!/K=!")
	check("cap:k=!", "cap:k", false, "K=!/K=*")
	check("cap:k=!", "cap:k=v", false, "K=!/K=v")

	// Instance K=*, Pattern variations
	check("cap:k", "cap:", true, "K=*/(none)")
	check("cap:k", "cap:k=?", true, "K=*/K=?")
	check("cap:k", "cap:k=!", false, "K=*/K=!")
	check("cap:k", "cap:k", true, "K=*/K=*")
	check("cap:k", "cap:k=v", true, "K=*/K=v")

	// Instance K=v, Pattern variations
	check("cap:k=v", "cap:", true, "K=v/(none)")
	check("cap:k=v", "cap:k=?", true, "K=v/K=?")
	check("cap:k=v", "cap:k=!", false, "K=v/K=!")
	check("cap:k=v", "cap:k", true, "K=v/K=*")
	check("cap:k=v", "cap:k=v", true, "K=v/K=v")
	check("cap:k=v", "cap:k=w", false, "K=v/K=w")
}

func TestMixedSpecialValues(t *testing.T) {
	// Test URNs with multiple special values
	pattern, err := NewTaggedUrnFromString("cap:required;optional=?;forbidden=!;exact=pdf")
	require.NoError(t, err)

	// Instance that satisfies all constraints
	goodInstance, _ := NewTaggedUrnFromString("cap:required=yes;optional=maybe;exact=pdf")
	matches, _ := goodInstance.Matches(pattern)
	assert.True(t, matches)

	// Instance missing required tag
	missingRequired, _ := NewTaggedUrnFromString("cap:optional=maybe;exact=pdf")
	matches, _ = missingRequired.Matches(pattern)
	assert.False(t, matches)

	// Instance has forbidden tag
	hasForbidden, _ := NewTaggedUrnFromString("cap:required=yes;forbidden=oops;exact=pdf")
	matches, _ = hasForbidden.Matches(pattern)
	assert.False(t, matches)

	// Instance with wrong exact value
	wrongExact, _ := NewTaggedUrnFromString("cap:required=yes;exact=doc")
	matches, _ = wrongExact.Matches(pattern)
	assert.False(t, matches)
}

func TestSerializationRoundTripSpecialValues(t *testing.T) {
	// All special values round-trip correctly
	originals := []string{
		"cap:ext=?",
		"cap:ext=!",
		"cap:ext", // * serializes as valueless
		"cap:a=?;b=!;c;d=exact",
	}

	for _, original := range originals {
		urn, err := NewTaggedUrnFromString(original)
		require.NoError(t, err, "Failed to parse: %s", original)
		serialized := urn.ToString()
		reparsed, err := NewTaggedUrnFromString(serialized)
		require.NoError(t, err, "Failed to reparse: %s", serialized)
		assert.True(t, urn.Equals(reparsed), "Round-trip failed for: %s", original)
	}
}

func TestCompatibilityWithSpecialValues(t *testing.T) {
	// ! is incompatible with * and specific values
	mustNot, _ := NewTaggedUrnFromString("cap:ext=!")
	mustHave, _ := NewTaggedUrnFromString("cap:ext=*")
	specific, _ := NewTaggedUrnFromString("cap:ext=pdf")
	unspecified, _ := NewTaggedUrnFromString("cap:ext=?")
	missing, _ := NewTaggedUrnFromString("cap:")

	compatible, _ := mustNot.IsCompatibleWith(mustHave)
	assert.False(t, compatible)

	compatible, _ = mustNot.IsCompatibleWith(specific)
	assert.False(t, compatible)

	compatible, _ = mustNot.IsCompatibleWith(unspecified)
	assert.True(t, compatible)

	compatible, _ = mustNot.IsCompatibleWith(missing)
	assert.True(t, compatible)

	compatible, _ = mustNot.IsCompatibleWith(mustNot)
	assert.True(t, compatible)

	// * is compatible with specific values
	compatible, _ = mustHave.IsCompatibleWith(specific)
	assert.True(t, compatible)

	compatible, _ = mustHave.IsCompatibleWith(mustHave)
	assert.True(t, compatible)

	// ? is compatible with everything
	compatible, _ = unspecified.IsCompatibleWith(mustNot)
	assert.True(t, compatible)

	compatible, _ = unspecified.IsCompatibleWith(mustHave)
	assert.True(t, compatible)

	compatible, _ = unspecified.IsCompatibleWith(specific)
	assert.True(t, compatible)

	compatible, _ = unspecified.IsCompatibleWith(unspecified)
	assert.True(t, compatible)

	compatible, _ = unspecified.IsCompatibleWith(missing)
	assert.True(t, compatible)
}

func TestSpecificityWithSpecialValues(t *testing.T) {
	// Verify graded specificity scoring
	exact, _ := NewTaggedUrnFromString("cap:a=x;b=y;c=z")        // 3*3 = 9
	mustHave, _ := NewTaggedUrnFromString("cap:a;b;c")           // 3*2 = 6
	mustNotUrn, _ := NewTaggedUrnFromString("cap:a=!;b=!;c=!")   // 3*1 = 3
	unspecified, _ := NewTaggedUrnFromString("cap:a=?;b=?;c=?")  // 3*0 = 0
	mixed, _ := NewTaggedUrnFromString("cap:a=x;b;c=!;d=?")      // 3+2+1+0 = 6

	assert.Equal(t, 9, exact.Specificity())
	assert.Equal(t, 6, mustHave.Specificity())
	assert.Equal(t, 3, mustNotUrn.Specificity())
	assert.Equal(t, 0, unspecified.Specificity())
	assert.Equal(t, 6, mixed.Specificity())

	// Test specificity tuples
	e, mha, mn := exact.SpecificityTuple()
	assert.Equal(t, 3, e)
	assert.Equal(t, 0, mha)
	assert.Equal(t, 0, mn)

	e, mha, mn = mustHave.SpecificityTuple()
	assert.Equal(t, 0, e)
	assert.Equal(t, 3, mha)
	assert.Equal(t, 0, mn)

	e, mha, mn = mustNotUrn.SpecificityTuple()
	assert.Equal(t, 0, e)
	assert.Equal(t, 0, mha)
	assert.Equal(t, 3, mn)

	e, mha, mn = unspecified.SpecificityTuple()
	assert.Equal(t, 0, e)
	assert.Equal(t, 0, mha)
	assert.Equal(t, 0, mn)

	e, mha, mn = mixed.SpecificityTuple()
	assert.Equal(t, 1, e)
	assert.Equal(t, 1, mha)
	assert.Equal(t, 1, mn)
}
