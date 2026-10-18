output "ecr_go_repository_url" {
  description = "Go の Lambda イメージを push する先"
  value       = aws_ecr_repository.go.repository_url
}
