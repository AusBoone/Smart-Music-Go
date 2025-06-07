terraform {
  required_version = ">= 1.5"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.region
}

resource "aws_ecs_cluster" "this" {
  name = "smart-music-go"
}

resource "aws_ecs_task_definition" "app" {
  family                   = "smart-music-go"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = 256
  memory                   = 512

  execution_role_arn = var.execution_role_arn
  task_role_arn      = var.task_role_arn

  container_definitions = jsonencode([
    {
      name  = "app"
      image = var.image
      portMappings = [
        {
          containerPort = 4000
          hostPort      = 4000
        }
      ]
      environment = [
        { name = "SPOTIFY_CLIENT_ID", value = var.spotify_client_id },
        { name = "SPOTIFY_CLIENT_SECRET", value = var.spotify_client_secret },
        { name = "SPOTIFY_REDIRECT_URL", value = var.spotify_redirect_url },
        { name = "DATABASE_PATH", value = "/data/smartmusic.db" }
      ]
      mountPoints = [
        {
          sourceVolume  = "data"
          containerPath = "/data"
        }
      ]
    }
  ])

  volume {
    name = "data"
    efs_volume_configuration {
      file_system_id = aws_efs_file_system.data.id
    }
  }
}

resource "aws_efs_file_system" "data" {
  creation_token = "smartmusic-data"
}

resource "aws_security_group" "lb" {
  name   = "smart-music-go-lb"
  vpc_id = var.vpc_id

  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_lb" "this" {
  name               = "smart-music-go"
  load_balancer_type = "application"
  subnets            = var.subnets
  security_groups    = [aws_security_group.lb.id]
}

resource "aws_lb_target_group" "app" {
  name     = "smart-music-go"
  port     = 4000
  protocol = "HTTP"
  vpc_id   = var.vpc_id
  health_check {
    path    = "/"
    matcher = "200"
  }
}

resource "aws_lb_listener" "http" {
  load_balancer_arn = aws_lb.this.arn
  port              = "80"
  protocol          = "HTTP"

  default_action {
    type = "redirect"
    redirect {
      port        = "443"
      protocol    = "HTTPS"
      status_code = "HTTP_301"
    }
  }
}

resource "aws_lb_listener" "https" {
  load_balancer_arn = aws_lb.this.arn
  port              = "443"
  protocol          = "HTTPS"
  ssl_policy        = "ELBSecurityPolicy-2016-08"
  certificate_arn   = var.certificate_arn

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.app.arn
  }
}

resource "aws_security_group" "service" {
  name   = "smart-music-go"
  vpc_id = var.vpc_id

  ingress {
    from_port       = 4000
    to_port         = 4000
    protocol        = "tcp"
    security_groups = [aws_security_group.lb.id]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_ecs_service" "app" {
  name            = "smart-music-go"
  cluster         = aws_ecs_cluster.this.id
  task_definition = aws_ecs_task_definition.app.arn
  desired_count   = 1
  launch_type     = "FARGATE"

  network_configuration {
    subnets         = var.subnets
    security_groups = [aws_security_group.service.id]
  }

  load_balancer {
    target_group_arn = aws_lb_target_group.app.arn
    container_name   = "app"
    container_port   = 4000
  }
}

output "load_balancer_dns" {
  value = aws_lb.this.dns_name
}
