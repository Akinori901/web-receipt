variable "project" {
  type        = string
  default     = "web-receipt"
  description = "リソース名の接頭辞"
}

variable "region" {
  type    = string
  default = "ap-northeast-1"
}

variable "keep_images" {
  type        = number
  default     = 20
  description = "ECR に残すイメージの数（ロールバックに使える世代数）"
}
