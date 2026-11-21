// sensitive_operations_test.go - Check the generated sensitive-output list against known cases

package cli

import (
	"strings"
	"testing"
)

// Operations the earlier hand-kept list covered must stay sensitive
func TestGeneratedListKeepsEarlierCoverage(t *testing.T) {
	for _, command := range []string{
		"codeartifact:get-authorization-token",
		"cognito-identity:get-credentials-for-identity",
		"cognito-identity:get-open-id-token",
		"cognito-identity:get-open-id-token-for-developer-identity",
		"connect:get-federation-token",
		"ec2:get-password-data",
		"ecr-public:get-authorization-token",
		"ecr:get-authorization-token",
		"ecr:get-login-password",
		"eks:get-token",
		"gamelift:get-instance-access",
		"lightsail:get-instance-access-details",
		"redshift-serverless:get-credentials",
		"redshift:get-cluster-credentials",
		"redshift:get-cluster-credentials-with-iam",
		"secretsmanager:batch-get-secret-value",
		"secretsmanager:get-random-password",
		"secretsmanager:get-secret-value",
		"sso:get-role-credentials",
		"ssm:get-parameter",
		"ssm:get-parameters",
		"ssm:get-parameters-by-path",
		"sts:get-federation-token",
		"sts:get-session-token",
	} {
		if !isSensitiveOutputOperation(command) {
			t.Errorf("%s is not sensitive", command)
		}
	}
}

// Secret-returning reads the earlier list missed are now sensitive, and ordinary reads are not
func TestGeneratedListSeparatesSecretsFromOrdinaryReads(t *testing.T) {
	for command, want := range map[string]bool{
		"cognito-idp:describe-user-pool-client":                  true,
		"ec2:describe-vpn-connections":                           true,
		"lightsail:get-relational-database-master-user-password": true,
		"sts:get-web-identity-token":                             true,
		"rds:describe-db-instances":                              false,
		"fsx:describe-file-systems":                              false,
		"ecs:describe-task-definition":                           false,
		"ec2:describe-instances":                                 false,
		"iam:list-users":                                         false,
		"s3api:list-buckets":                                     false,
		"sts:get-caller-identity":                                false,
	} {
		if got := isSensitiveOutputOperation(command); got != want {
			t.Errorf("isSensitiveOutputOperation(%q) = %t, want %t", command, got, want)
		}
	}
}

// Generated entries must be valid policy rules for read-style operations
func TestGeneratedListEntriesAreReadStyleRules(t *testing.T) {
	if generatedFromBotocore == "" || len(generatedSensitiveOperations) == 0 {
		t.Fatal("generated list is empty")
	}
	for command := range generatedSensitiveOperations {
		if err := validatePolicyRule(command); err != nil {
			t.Errorf("%s: %v", command, err)
		}
		_, operation, _ := splitCommand(command)
		if !isReadOperation(operation) {
			t.Errorf("%s is not a read-style operation", command)
		}
	}
	for _, pattern := range manualSensitivePatterns {
		if err := validatePolicyRule(pattern); err != nil {
			t.Errorf("%s: %v", pattern, err)
		}
	}
}

// splitCommand separates service and operation
func splitCommand(command string) (string, string, bool) {
	return strings.Cut(command, ":")
}
