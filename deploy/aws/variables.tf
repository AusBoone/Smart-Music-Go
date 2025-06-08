variable "region" {
  description = "AWS region"
  type        = string
}

variable "vpc_id" {
  description = "VPC id to deploy into"
  type        = string
}

variable "subnets" {
  description = "List of subnets for the service"
  type        = list(string)
}

variable "certificate_arn" {
  description = "ACM certificate for HTTPS"
  type        = string
}

variable "execution_role_arn" {
  description = "IAM role for ECS task execution"
  type        = string
}

variable "task_role_arn" {
  description = "IAM role for the task"
  type        = string
}

variable "image" {
  description = "Container image URI"
  type        = string
}

variable "spotify_client_id" {
  description = "Spotify client ID"
  type        = string
  sensitive   = true
}

variable "spotify_client_secret" {
  description = "Spotify client secret"
  type        = string
  sensitive   = true
}

variable "spotify_redirect_url" {
  description = "OAuth redirect URL"
  type        = string
  sensitive   = true
}
