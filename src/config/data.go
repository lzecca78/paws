package config

type AwsGetCallerIdentitySpec struct {
	Account string `mapstructure:"account" json:"account" yaml:"account"`
	ARN     string `mapstructure:"arn" json:"arn" yaml:"arn"`
	UserID  string `mapstructure:"user_id" json:"user_id" yaml:"user_id"`
}
