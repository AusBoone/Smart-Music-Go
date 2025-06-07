# Deployment Guide

This directory contains example Terraform configuration for deploying the Docker
image to AWS using ECS Fargate and an Application Load Balancer. The configuration
assumes that a VPC, subnets and IAM roles already exist.

## Steps

1. Build and push the Docker image to Amazon ECR.
2. Fill out the required variables in `terraform.tfvars` or pass them on the command line.
3. Run `terraform init` followed by `terraform apply`.
4. The output `load_balancer_dns` contains the public address of the service.

Environment variables such as the Spotify credentials are passed directly to the
container in the task definition. Update the variables in `terraform.tfvars` to
match your environment.

SSL is terminated at the load balancer using an ACM certificate specified by
`certificate_arn`.
