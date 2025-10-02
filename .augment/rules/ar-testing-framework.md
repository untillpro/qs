# .augment/rules/qs-testing-framework.md
type: auto
description: Testing guidelines for qs project

- Use testify framework for unit tests (github.com/stretchr/testify/require)
- Write system tests in sys_test.go for new commands
- Test with real GitHub repositories using systrun framework
- Include both positive and negative test cases