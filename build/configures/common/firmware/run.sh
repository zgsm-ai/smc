#!/bin/bash
set -e
set -u
set -o pipefail 2>/dev/null || true

# -------------------------- Initialize Configuration --------------------------
SCRIPT_NAME=$(basename "$0")
LOG_FILE="${SCRIPT_NAME%.*}.log"
exec > >(tee -a "$LOG_FILE") 2>&1

# -------------------------- Constants Definition --------------------------
# Get the machine's IP
SERVER_IP=$(hostname -I | awk '{ print $1 }')
declare -r SERVER_IP

# Get the current directory
declare -r BASE_DIR=$(pwd)

# -------------------------- Function Definitions --------------------------
docker-compose() {
    # Check if docker has compose subcommand
    if docker compose version >/dev/null 2>&1; then
        command docker compose "$@"
    else
        command docker-compose "$@"
    fi
}

log() {
    local level=$1
    local message=$2
    local timestamp=$(date "+%Y-%m-%d %H:%M:%S")
    echo -e "[${timestamp}] [${level}] ${message}"
}

get_image_envs() {
    local target_base="${BASE_DIR}"
    local output_file="${target_base}/.images.env"
    
    # 切换到目标目录
    cd "$target_base" || {
        log "ERROR" "无法切换到目录: $target_base"
        return 1
    }
    
    log "INFO" "开始收集镜像环境配置..."
    
    # 清空或创建输出文件
    > "$output_file"
    
    # 查找所有 image.env 文件并合并内容
    local found_files=0
    while IFS= read -r env_file; do
        found_files=$((found_files + 1))
        log "INFO" "处理镜像配置文件: $env_file"
        
        # 将非空、非注释行追加到输出文件
        grep -v '^#' "$env_file" | grep -v '^$' >> "$output_file"
    done < <(find . -name "image.env" -type f)
    
    if [[ $found_files -eq 0 ]]; then
        log "WARN" "未找到任何 image.env 文件"
    else
        log "INFO" "已收集 ${found_files} 个镜像配置文件"
        log "INFO" "镜像环境配置已合并到: $output_file"
    fi
    
    # 切换回原目录
    cd - >/dev/null
    return 0
}

merge_env() {
    local target="$1"
    local source="$2"
    
    # 检查 source 文件是否存在
    if [[ ! -f "$source" ]]; then
        log "ERROR" "源文件不存在: $source"
        return 1
    fi
    
    # 检查 target 文件是否存在，不存在则创建
    if [[ ! -f "$target" ]]; then
        touch "$target"
    fi
    
    log "INFO" "开始合并环境配置: $source -> $target"
    
    # 读取 source 文件的非注释行和非空行，追加到 target 文件
    local line_count=0
    local line
    while IFS= read -r line || [[ -n "$line" ]]; do
        # 跳过注释行（以 # 开头，包括前面有空格的情况）
        if [[ "$line" =~ ^[[:space:]]*# ]]; then
            continue
        fi
        # 跳过纯空格的行
        if [[ -z "${line// /}" ]]; then
            continue
        fi
        
        # 追加到 target 文件
        echo "$line" >> "$target"
        line_count=$((line_count + 1))
    done < "$source"
    
    log "INFO" "已追加 ${line_count} 行到: $target"
    return 0
}

create_dot_env() {
    log "INFO" "生成镜像环境配置文件..."
    if ! get_image_envs; then
        log "ERROR" "生成镜像环境配置失败"
        return 1
    fi
    # 清空 .env 文件（如果存在）
    > .env 2>/dev/null || :  # 使用 : 确保命令总是成功
    # 调用 merge_env，将 ".images.env" 和 "costrict.env" 合并到 ".env" 文件中
    merge_env .env .images.env
    merge_env .env costrict.env
    
    # source 加载 costrict.env
    if [ -f "costrict.env" ]; then
        source costrict.env
    fi
    
    # 判断 COSTRICT_HOST 是否已定义，若未定义则添加到 .env
    if [ -z "${COSTRICT_HOST:-}" ]; then
        COSTRICT_HOST="${SERVER_IP}"
        echo "COSTRICT_HOST=\"${SERVER_IP}\"" >> .env
    fi
    
    # 判断 COSTRICT_BASEURL 是否已定义，若未定义则添加到 .env
    if [ -z "${COSTRICT_BASEURL:-}" ]; then
        echo "COSTRICT_BASEURL=\"http://${COSTRICT_HOST}:${PORT_APISIX_ENTRY}\"" >> .env
    fi
    
    log "INFO" ".env 文件创建完成"
    return 0
}

check_core_dependencies() {
    local missing_deps=()
    local base_path="${BASE_DIR}"
    
    # 检查依赖项目录/文件
    local deps=(
        ".env"
        ".images.env"
        "docker-compose.yml"
        "costrict-admin-backend"
        "apisix"
        "portal"
        "portal/data/costrict-admin"
        "portal/data/costrict-admin/index.html"
    )
    
    for dep in "${deps[@]}"; do
        local dep_path="${base_path}/${dep}"
        if [[ ! -e "$dep_path" ]]; then
            missing_deps+=("$dep")
            log "WARN" "缺失依赖: ${dep}"
        fi
    done
    
    if [[ ${#missing_deps[@]} -gt 0 ]]; then
        log "ERROR" "核心依赖检查失败，以下依赖项不存在:"
        for dep in "${missing_deps[@]}"; do
            log "ERROR" "  - ${dep}"
        done
        return 1
    fi
    
    log "INFO" "核心依赖检查通过"
    return 0
}

start_docker_services() {
    log "INFO" "生成环境变量文件.env..."
    if ! create_dot_env; then
        log "ERROR" "生成环境变量文件.env失败"
        return 1
    fi
    
    log "INFO" "检查核心依赖..."
    if ! check_core_dependencies; then
        log "ERROR" "核心依赖检查失败，请确保所有依赖目录和文件都存在"
        return 1
    fi
    
    log "INFO" "启动Docker Compose服务..."
    if ! docker-compose  -f docker-compose.yml up -d; then
        log "ERROR" "Docker Compose服务启动失败"
        return 1
    fi
    log "INFO" "Docker Compose服务启动完成"
    return 0
}

stop_docker_services() {
    log "INFO" "停止Docker Compose服务..."
    if ! docker-compose  -f docker-compose.yml down; then
        log "ERROR" "Docker Compose服务停止失败"
        return 1
    fi
    log "INFO" "Docker Compose服务已停止"
    return 0
}

check_docker_status() {
    log "INFO" "检查Docker Compose服务状态..."
    # 获取docker-compose ps的输出
    local ps_output
    ps_output=$(docker-compose  -f docker-compose.yml ps 2>&1)
    local ps_exit_code=$?
    
    # 如果命令执行失败，说明服务未运行
    if [[ $ps_exit_code -ne 0 ]]; then
        log "INFO" "Docker Compose服务未运行"
        return 1
    fi
    
    # 逐行处理统计容器状态
    local healthy_count=0 unhealthy_count=0 starting_count=0 no_status_count=0
    local line_count=0
    
    # 逐行处理容器信息，跳过第一行（表头）
    while IFS= read -r line; do
        ((line_count++))
        # 跳过第一行表头
        [[ $line_count -eq 1 ]] && continue
        
        # 跳过空行
        [[ -z "$line" ]] && continue
        
        #根据状态标识分类统计
        if [[ "$line" == *"(health: starting)"* ]]; then
            ((starting_count++))
        elif [[ "$line" == *"(healthy)"* ]]; then
            ((healthy_count++))
        elif [[ "$line" == *"(unhealthy)"* ]]; then
            ((unhealthy_count++))
        else
            # 无状态标识的容器
            ((no_status_count++))
        fi
    done <<< "$ps_output"
    
    # 计算总容器数
    local total_count=$((healthy_count + unhealthy_count + starting_count + no_status_count))
    
    # 显示四类统计结果
    log "INFO" "容器状态统计: (总计 ${total_count} 个)"
    log "INFO" "  - 健康(healthy): ${healthy_count} 个"
    log "INFO" "  - 不健康(unhealthy): ${unhealthy_count} 个"
    log "INFO" "  - 健康检查中(health: starting): ${starting_count} 个"
    log "INFO" "  - 无状态标识(no status): ${no_status_count} 个"
    
    # 如果有不健康的容器，显示详细信息
    if [[ $unhealthy_count -gt 0 ]]; then
        # 显示不健康的容器
        log "WARN" "不健康容器详情:"
        echo "$ps_output" | while IFS= read -r line; do
            if [[ "$line" == *"(unhealthy)"* ]]; then
                local container_name
                container_name=$(echo "$line" | awk '{print $1}')
                log "WARN" "  - ${container_name}"
            fi
        done
    fi
    
    # 如果有正在健康检查的容器，显示详细信息
    if [[ $starting_count -gt 0 ]]; then
        # 显示健康检查中的容器
        log "INFO" "健康检查中容器详情:"
        echo "$ps_output" | while IFS= read -r line; do
            if [[ "$line" == *"(health: starting)"* ]]; then
                local container_name
                container_name=$(echo "$line" | awk '{print $1}')
                log "INFO" "  - ${container_name}"
            fi
        done
    fi
    
    # 如果有容器在运行（排除退出的），则认为服务正在运行
    local running_count=$((healthy_count + unhealthy_count + starting_count + no_status_count))
    if [[ $running_count -gt 0 ]]; then
        log "INFO" "Docker Compose服务正在运行"
        return 0
    else
        log "INFO" "Docker Compose服务未运行"
        return 1
    fi
}

# -------------------------- Command Functions --------------------------
cmd_start() {
    log "INFO" "开始启动系统..."
    
    # 启动Docker Compose服务
    if ! start_docker_services; then
        return 1
    fi
    
    log "INFO" "系统启动完成"
    log "INFO" "请登录到诸葛神码后端管理页面 [http://${SERVER_IP}:39080/costrict-admin] (默认账号: admin, 密码: admin)"
    return 0
}

cmd_stop() {
    log "INFO" "开始停止系统..."
    
    # 停止Docker Compose服务
    if ! stop_docker_services; then
        return 1
    fi
    
    log "INFO" "系统已停止"
    return 0
}

cmd_restart() {
    log "INFO" "开始重启系统..."
    
    # 停止Docker Compose服务
    stop_docker_services
    
    sleep 3
    
    # 启动Docker Compose服务（跳过首次初始化）
    if ! start_docker_services; then
        return 1
    fi
    
    log "INFO" "系统重启完成"
    log "INFO" "请登录到诸葛神码后端管理页面 [http://${SERVER_IP}:39080/costrict-admin] (默认账号: admin, 密码: admin)"
    return 0
}

cmd_status() {
    log "INFO" "系统状态："
    log "INFO" "=========="
    
    # 检查Docker Compose服务状态
    if check_docker_status; then
        log "INFO" "Docker Compose服务: 运行中"
    else
        log "INFO" "Docker Compose服务: 未运行"
    fi
    
    log "INFO" "=========="
    return 0
}

cmd_details() {
    docker-compose  -f docker-compose.yml ps
}

# -------------------------- Usage --------------------------
show_usage() {
    cat << EOF
用法: $0 [选项]

选项:
    start              启动系统（默认）
    stop               停止系统
    restart            重启系统
    status             查看系统状态
    detail             查看容器详细信息
    -h, --help         显示帮助信息

首次运行前需先执行 init.sh 进行系统初始化。

示例:
    $0 start           # 启动系统
    $0 stop            # 停止系统
    $0 restart         # 重启系统
    $0 status          # 查看状态
    $0 detail          # 查看容器详细信息

EOF
}

# -------------------------- Main Logic --------------------------
main() {
    local action="${1:-start}"
    
    log "INFO" "脚本启动，日志文件: $LOG_FILE"
    
    case "$action" in
        start)
            cmd_start
            ;;
        stop)
            cmd_stop
            ;;
        restart)
            cmd_restart
            ;;
        status)
            cmd_status
            ;;
        detail)
            cmd_details
            ;;
        -h|--help)
            show_usage
            exit 0
            ;;
        *)
            log "ERROR" "未知参数: $action"
            show_usage
            exit 1
            ;;
    esac
}

main "$@"