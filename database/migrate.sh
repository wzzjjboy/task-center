#!/bin/bash
# ========================================
# Task Center golang-migrate 管理脚本
# ========================================
# 版本: 2.0
# 迁移工具: golang-migrate
# 描述: 基于 golang-migrate 的数据库迁移管理工具

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 配置变量
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MIGRATIONS_DIR="$SCRIPT_DIR/migrations"

# 数据库配置
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-3306}"
DB_NAME="${DB_NAME:-task_center}"
DB_USER="${DB_USER:-root}"
DB_PASSWORD="${DB_PASSWORD:-root123}"
DOCKER_CONTAINER="${DOCKER_CONTAINER:-}"

# 构建数据库连接字符串
if [ -n "$DOCKER_CONTAINER" ]; then
    # 使用 Docker 容器时的特殊处理
    DATABASE_URL="mysql://$DB_USER:$DB_PASSWORD@tcp(localhost:$DB_PORT)/$DB_NAME"
else
    DATABASE_URL="mysql://$DB_USER:$DB_PASSWORD@tcp($DB_HOST:$DB_PORT)/$DB_NAME"
fi

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查依赖
check_dependencies() {
    if ! command -v migrate &> /dev/null; then
        log_error "golang-migrate 未安装。请运行以下命令安装："
        echo "go install -tags 'mysql' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"
        exit 1
    fi

    if [ ! -d "$MIGRATIONS_DIR" ]; then
        log_error "迁移目录不存在: $MIGRATIONS_DIR"
        exit 1
    fi
}

# 测试数据库连接
test_connection() {
    log_info "测试数据库连接..."

    if [ -n "$DOCKER_CONTAINER" ]; then
        if ! docker exec "$DOCKER_CONTAINER" mysql -u"$DB_USER" -p"$DB_PASSWORD" -e "SELECT 1;" >/dev/null 2>&1; then
            log_error "无法连接到 Docker 容器中的数据库: $DOCKER_CONTAINER"
            exit 1
        fi
    else
        if ! mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" -p"$DB_PASSWORD" -e "SELECT 1;" >/dev/null 2>&1; then
            log_error "无法连接到数据库: $DB_HOST:$DB_PORT"
            exit 1
        fi
    fi

    log_success "数据库连接成功"
}

# 显示帮助信息
show_help() {
    echo "Task Center 数据库迁移管理工具 (golang-migrate)"
    echo ""
    echo "用法: $0 <命令> [参数]"
    echo ""
    echo "命令:"
    echo "  up [N]           执行迁移 (可选: 指定执行N个迁移)"
    echo "  down N           回滚N个迁移"
    echo "  goto VERSION     迁移到指定版本"
    echo "  drop             删除所有数据和迁移历史 (危险操作)"
    echo "  force VERSION    强制设置迁移版本 (修复脏状态)"
    echo "  version          显示当前迁移版本"
    echo "  status           显示迁移状态和历史"
    echo "  create NAME      创建新的迁移文件"
    echo "  validate         验证迁移文件"
    echo "  help             显示此帮助信息"
    echo ""
    echo "环境变量:"
    echo "  DB_HOST          数据库主机 (默认: localhost)"
    echo "  DB_PORT          数据库端口 (默认: 3306)"
    echo "  DB_USER          数据库用户 (默认: root)"
    echo "  DB_PASSWORD      数据库密码 (默认: root123)"
    echo "  DB_NAME          数据库名称 (默认: task_center)"
    echo "  DOCKER_CONTAINER Docker容器名称 (可选)"
    echo ""
    echo "示例:"
    echo "  $0 up                    # 执行所有待执行的迁移"
    echo "  $0 up 2                  # 执行接下来的2个迁移"
    echo "  $0 down 1                # 回滚最新的1个迁移"
    echo "  $0 goto 3                # 迁移到版本3"
    echo "  $0 create add_user_index # 创建新迁移文件"
    echo "  $0 status                # 查看迁移状态"
}

# 执行迁移命令
run_migrate() {
    local cmd="$1"
    shift

    log_info "执行迁移命令: migrate $cmd $*"

    if migrate -database "$DATABASE_URL" -path "$MIGRATIONS_DIR" "$cmd" "$@"; then
        log_success "迁移命令执行成功"
        return 0
    else
        log_error "迁移命令执行失败"
        return 1
    fi
}

# 显示迁移状态
show_status() {
    log_info "=== 数据库迁移状态 ==="
    echo ""

    # 获取当前版本
    local current_version
    current_version=$(migrate -database "$DATABASE_URL" -path "$MIGRATIONS_DIR" version 2>/dev/null || echo "unknown")
    echo "当前迁移版本: $current_version"
    echo ""

    # 列出所有迁移文件
    echo "可用的迁移文件:"
    if [ -d "$MIGRATIONS_DIR" ]; then
        ls -1 "$MIGRATIONS_DIR"/*.up.sql 2>/dev/null | sed 's/.*\///' | sed 's/\.up\.sql$//' | sort -V || echo "未找到迁移文件"
    fi
    echo ""

    # 显示迁移历史 (如果有 schema_migrations 表)
    if [ -n "$DOCKER_CONTAINER" ]; then
        docker exec "$DOCKER_CONTAINER" mysql -u"$DB_USER" -p"$DB_PASSWORD" "$DB_NAME" -e "
            SELECT version, dirty,
                   CASE WHEN dirty = 0 THEN 'SUCCESS' ELSE 'FAILED' END as status
            FROM schema_migrations ORDER BY version;
        " 2>/dev/null | head -20 || echo "迁移历史表不存在或无法访问"
    else
        mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" -p"$DB_PASSWORD" "$DB_NAME" -e "
            SELECT version, dirty,
                   CASE WHEN dirty = 0 THEN 'SUCCESS' ELSE 'FAILED' END as status
            FROM schema_migrations ORDER BY version;
        " 2>/dev/null | head -20 || echo "迁移历史表不存在或无法访问"
    fi
}

# 创建新迁移
create_migration() {
    local name="$1"
    if [ -z "$name" ]; then
        log_error "请提供迁移名称"
        echo "用法: $0 create <migration_name>"
        exit 1
    fi

    log_info "创建新迁移: $name"
    cd "$MIGRATIONS_DIR"
    migrate create -ext sql -dir . -seq "$name"
    log_success "迁移文件创建完成"
    ls -la *"$name"*
}

# 验证迁移文件
validate_migrations() {
    log_info "验证迁移文件..."

    local up_files
    local down_files
    up_files=$(ls -1 "$MIGRATIONS_DIR"/*.up.sql 2>/dev/null | wc -l)
    down_files=$(ls -1 "$MIGRATIONS_DIR"/*.down.sql 2>/dev/null | wc -l)

    if [ "$up_files" -ne "$down_files" ]; then
        log_warning "up 文件数量 ($up_files) 与 down 文件数量 ($down_files) 不匹配"
    fi

    log_info "找到 $up_files 个 up 迁移文件和 $down_files 个 down 迁移文件"

    # 检查语法 (简单检查)
    for file in "$MIGRATIONS_DIR"/*.sql; do
        if [ -f "$file" ]; then
            if ! grep -q "CREATE\|ALTER\|DROP\|INSERT\|UPDATE\|DELETE" "$file" 2>/dev/null; then
                log_warning "文件可能为空或不包含SQL语句: $(basename "$file")"
            fi
        fi
    done

    log_success "迁移文件验证完成"
}

# 主函数
main() {
    local command="${1:-help}"

    case "$command" in
        "up")
            check_dependencies
            test_connection
            shift
            run_migrate up "$@"
            ;;
        "down")
            check_dependencies
            test_connection
            if [ -z "$2" ]; then
                log_error "down 命令需要指定回滚数量"
                echo "用法: $0 down <number>"
                exit 1
            fi
            run_migrate down "$2"
            ;;
        "goto")
            check_dependencies
            test_connection
            if [ -z "$2" ]; then
                log_error "goto 命令需要指定版本号"
                echo "用法: $0 goto <version>"
                exit 1
            fi
            run_migrate goto "$2"
            ;;
        "drop")
            check_dependencies
            test_connection
            log_warning "这将删除所有数据和迁移历史！"
            read -p "确认继续? (y/N): " -n 1 -r
            echo
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                run_migrate drop
            else
                log_info "操作已取消"
            fi
            ;;
        "force")
            check_dependencies
            test_connection
            if [ -z "$2" ]; then
                log_error "force 命令需要指定版本号"
                echo "用法: $0 force <version>"
                exit 1
            fi
            run_migrate force "$2"
            ;;
        "version")
            check_dependencies
            test_connection
            run_migrate version
            ;;
        "status")
            check_dependencies
            test_connection
            show_status
            ;;
        "create")
            check_dependencies
            create_migration "$2"
            ;;
        "validate")
            validate_migrations
            ;;
        "help"|"-h"|"--help")
            show_help
            ;;
        *)
            log_error "未知命令: $command"
            echo ""
            show_help
            exit 1
            ;;
    esac
}

# 执行主函数
main "$@"