# VPC Module - Creates or uses existing VPC
# Smart: Creates new VPC if vpc_id is not provided

locals {
  create_vpc = var.vpc_id == ""
  vpc_id     = local.create_vpc ? aws_vpc.main[0].id : var.vpc_id
  
  # Calculate subnet CIDRs from VPC CIDR
  azs = slice(data.aws_availability_zones.available.names, 0, var.az_count)
}

data "aws_availability_zones" "available" {
  state = "available"
}

# Lookup existing VPC if provided
data "aws_vpc" "existing" {
  count = local.create_vpc ? 0 : 1
  id    = var.vpc_id
}

data "aws_subnets" "existing_private" {
  count = local.create_vpc ? 0 : 1
  
  filter {
    name   = "vpc-id"
    values = [var.vpc_id]
  }
  
  tags = {
    Tier = "Private"
  }
}

data "aws_subnets" "existing_public" {
  count = local.create_vpc ? 0 : 1
  
  filter {
    name   = "vpc-id"
    values = [var.vpc_id]
  }
  
  tags = {
    Tier = "Public"
  }
}

#------------------------------------------------------------------------------
# VPC
#------------------------------------------------------------------------------
resource "aws_vpc" "main" {
  count = local.create_vpc ? 1 : 0

  cidr_block           = var.vpc_cidr
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = merge(var.tags, {
    Name = "${var.project_name}-${var.environment}-vpc"
  })
}

#------------------------------------------------------------------------------
# Internet Gateway
#------------------------------------------------------------------------------
resource "aws_internet_gateway" "main" {
  count = local.create_vpc ? 1 : 0

  vpc_id = aws_vpc.main[0].id

  tags = merge(var.tags, {
    Name = "${var.project_name}-${var.environment}-igw"
  })
}

#------------------------------------------------------------------------------
# Public Subnets
#------------------------------------------------------------------------------
resource "aws_subnet" "public" {
  count = local.create_vpc ? var.az_count : 0

  vpc_id                  = aws_vpc.main[0].id
  cidr_block              = cidrsubnet(var.vpc_cidr, 4, count.index)
  availability_zone       = local.azs[count.index]
  map_public_ip_on_launch = true

  tags = merge(var.tags, {
    Name = "${var.project_name}-${var.environment}-public-${local.azs[count.index]}"
    Tier = "Public"
  })
}

#------------------------------------------------------------------------------
# Private Subnets
#------------------------------------------------------------------------------
resource "aws_subnet" "private" {
  count = local.create_vpc ? var.az_count : 0

  vpc_id            = aws_vpc.main[0].id
  cidr_block        = cidrsubnet(var.vpc_cidr, 4, count.index + var.az_count)
  availability_zone = local.azs[count.index]

  tags = merge(var.tags, {
    Name = "${var.project_name}-${var.environment}-private-${local.azs[count.index]}"
    Tier = "Private"
  })
}

#------------------------------------------------------------------------------
# NAT Gateway (one per AZ for HA in production)
#------------------------------------------------------------------------------
resource "aws_eip" "nat" {
  count = local.create_vpc && var.enable_nat_gateway ? (var.single_nat_gateway ? 1 : var.az_count) : 0

  domain = "vpc"

  tags = merge(var.tags, {
    Name = "${var.project_name}-${var.environment}-nat-eip-${count.index + 1}"
  })

  depends_on = [aws_internet_gateway.main]
}

resource "aws_nat_gateway" "main" {
  count = local.create_vpc && var.enable_nat_gateway ? (var.single_nat_gateway ? 1 : var.az_count) : 0

  allocation_id = aws_eip.nat[count.index].id
  subnet_id     = aws_subnet.public[count.index].id

  tags = merge(var.tags, {
    Name = "${var.project_name}-${var.environment}-nat-${count.index + 1}"
  })

  depends_on = [aws_internet_gateway.main]
}

#------------------------------------------------------------------------------
# Route Tables
#------------------------------------------------------------------------------
resource "aws_route_table" "public" {
  count = local.create_vpc ? 1 : 0

  vpc_id = aws_vpc.main[0].id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.main[0].id
  }

  tags = merge(var.tags, {
    Name = "${var.project_name}-${var.environment}-public-rt"
  })
}

resource "aws_route_table" "private" {
  count = local.create_vpc ? (var.single_nat_gateway ? 1 : var.az_count) : 0

  vpc_id = aws_vpc.main[0].id

  dynamic "route" {
    for_each = var.enable_nat_gateway ? [1] : []
    content {
      cidr_block     = "0.0.0.0/0"
      nat_gateway_id = var.single_nat_gateway ? aws_nat_gateway.main[0].id : aws_nat_gateway.main[count.index].id
    }
  }

  tags = merge(var.tags, {
    Name = "${var.project_name}-${var.environment}-private-rt-${count.index + 1}"
  })
}

resource "aws_route_table_association" "public" {
  count = local.create_vpc ? var.az_count : 0

  subnet_id      = aws_subnet.public[count.index].id
  route_table_id = aws_route_table.public[0].id
}

resource "aws_route_table_association" "private" {
  count = local.create_vpc ? var.az_count : 0

  subnet_id      = aws_subnet.private[count.index].id
  route_table_id = var.single_nat_gateway ? aws_route_table.private[0].id : aws_route_table.private[count.index].id
}

#------------------------------------------------------------------------------
# VPC Endpoints (for cost optimization and security)
#------------------------------------------------------------------------------
resource "aws_vpc_endpoint" "dynamodb" {
  count = local.create_vpc && var.enable_vpc_endpoints ? 1 : 0

  vpc_id            = aws_vpc.main[0].id
  service_name      = "com.amazonaws.${var.aws_region}.dynamodb"
  vpc_endpoint_type = "Gateway"

  route_table_ids = concat(
    [aws_route_table.public[0].id],
    aws_route_table.private[*].id
  )

  tags = merge(var.tags, {
    Name = "${var.project_name}-${var.environment}-dynamodb-endpoint"
  })
}

resource "aws_vpc_endpoint" "ecr_api" {
  count = local.create_vpc && var.enable_vpc_endpoints ? 1 : 0

  vpc_id              = aws_vpc.main[0].id
  service_name        = "com.amazonaws.${var.aws_region}.ecr.api"
  vpc_endpoint_type   = "Interface"
  private_dns_enabled = true

  subnet_ids         = aws_subnet.private[*].id
  security_group_ids = [aws_security_group.vpc_endpoints[0].id]

  tags = merge(var.tags, {
    Name = "${var.project_name}-${var.environment}-ecr-api-endpoint"
  })
}

resource "aws_vpc_endpoint" "ecr_dkr" {
  count = local.create_vpc && var.enable_vpc_endpoints ? 1 : 0

  vpc_id              = aws_vpc.main[0].id
  service_name        = "com.amazonaws.${var.aws_region}.ecr.dkr"
  vpc_endpoint_type   = "Interface"
  private_dns_enabled = true

  subnet_ids         = aws_subnet.private[*].id
  security_group_ids = [aws_security_group.vpc_endpoints[0].id]

  tags = merge(var.tags, {
    Name = "${var.project_name}-${var.environment}-ecr-dkr-endpoint"
  })
}

resource "aws_vpc_endpoint" "logs" {
  count = local.create_vpc && var.enable_vpc_endpoints ? 1 : 0

  vpc_id              = aws_vpc.main[0].id
  service_name        = "com.amazonaws.${var.aws_region}.logs"
  vpc_endpoint_type   = "Interface"
  private_dns_enabled = true

  subnet_ids         = aws_subnet.private[*].id
  security_group_ids = [aws_security_group.vpc_endpoints[0].id]

  tags = merge(var.tags, {
    Name = "${var.project_name}-${var.environment}-logs-endpoint"
  })
}

resource "aws_vpc_endpoint" "secretsmanager" {
  count = local.create_vpc && var.enable_vpc_endpoints ? 1 : 0

  vpc_id              = aws_vpc.main[0].id
  service_name        = "com.amazonaws.${var.aws_region}.secretsmanager"
  vpc_endpoint_type   = "Interface"
  private_dns_enabled = true

  subnet_ids         = aws_subnet.private[*].id
  security_group_ids = [aws_security_group.vpc_endpoints[0].id]

  tags = merge(var.tags, {
    Name = "${var.project_name}-${var.environment}-secretsmanager-endpoint"
  })
}

#------------------------------------------------------------------------------
# Security Groups
#------------------------------------------------------------------------------
resource "aws_security_group" "vpc_endpoints" {
  count = local.create_vpc && var.enable_vpc_endpoints ? 1 : 0

  name        = "${var.project_name}-${var.environment}-vpc-endpoints-sg"
  description = "Security group for VPC endpoints"
  vpc_id      = aws_vpc.main[0].id

  ingress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = [var.vpc_cidr]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = merge(var.tags, {
    Name = "${var.project_name}-${var.environment}-vpc-endpoints-sg"
  })
}
