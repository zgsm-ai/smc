#!/bin/bash

log() {
    local level=$1
    local message=$2
    local timestamp=$(date "+%Y-%m-%d %H:%M:%S")
    echo -e "[${timestamp}] [${level}] ${message}"
}

get_image_envs() {
    local output_file="$1"
    
    # 切换到目标目录
    cd "${base_dir}" || {
        log "ERROR" "无法切换到目录: ${base_dir}"
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
        log "INFO" "已收集 ${found_files} 个镜像环境配置文件"
        log "INFO" "镜像环境配置已合并到: $output_file"
    fi
    
    # 切换回原目录
    cd - >/dev/null
    return 0
}

# 使用getopt解析参数
TEMP=$(getopt -o o:f: --long output:,from: -n "$0" -- "$@")
eval set -- "$TEMP"

# 默认值
output_dir="."
base_dir=$(pwd)

# 解析参数
while true ; do
    case "$1" in
        -o|--output)
            output_dir="$2"
            shift 2
            ;;
        -f|--from)
            base_dir="$2"
            shift 2
            ;;
        --) shift ; break ;;
        *) echo "参数解析错误" >&2 ; exit 1 ;;
    esac
done

# 确保output_dir以'/'结尾
[[ "${output_dir: -1}" != "/" ]] && output_dir="${output_dir}/"

envs_file="${output_dir}.images.env"
list_file="${output_dir}.images.list"

mkdir -p ${output_dir}

get_image_envs ${envs_file} || {
    log "ERROR" "获取镜像环境配置失败"
    exit 1
}

# 从.images.env提取镜像列表
log "INFO" "生成镜像列表文件: $list_file"
awk -F'=' '{print $2}' "${envs_file}" > "${list_file}"
