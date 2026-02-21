#!/usr/bin/env bash
set -Eeuo pipefail

# uspecs automation
#
# Usage:
#   uspecs change new <change-name> [--issue-url <url>] [--branch]
#   uspecs change archive <change-folder-name>
#
# change new:
#   Creates Change Folder and change.md with frontmatter:
#     - Folder: <changes_folder from conf.md>/ymdHM-<change-name>
#     - registered_at: YYYY-MM-DDTHH:MM:SSZ
#     - change_id: ymdHM-<change-name>
#     - baseline: <commit-hash> (if git repository)
#     - issue_url: <url> (if --issue-url provided)
#   Creates git branch (if --branch provided and git repository exists)
#   Prints: <relative-path-to-change-folder> (e.g. uspecs/changes/2602201746-my-change)
#
# change archive:
#   Archives change folder to <changes-folder>/archive/yymm/ymdHM-<change-name>
#   Adds archived_at metadata and updates folder date prefix

error() {
    echo "Error: $1" >&2
    exit 1
}

get_timestamp() {
    date -u +"%Y-%m-%dT%H:%M:%SZ"
}

get_baseline() {
    if git rev-parse --git-dir > /dev/null 2>&1; then
        git rev-parse HEAD 2>/dev/null || echo ""
    else
        echo ""
    fi
}

get_folder_name() {
    local path="$1"
    basename "$path"
}

count_uncompleted_items() {
    local folder="$1"
    local count
    count=$(grep -r "^\s*-\s*\[ \]" "$folder"/*.md 2>/dev/null | wc -l)
    echo "${count:-0}" | tr -d ' '
}

extract_change_name() {
    local folder_name="$1"
    echo "$folder_name" | sed 's/^[0-9]\{10\}-//'
}

move_folder() {
    local source="$1"
    local destination="$2"
    if git rev-parse --git-dir > /dev/null 2>&1; then
        git mv "$source" "$destination" 2>/dev/null || mv "$source" "$destination"
    else
        mv "$source" "$destination"
    fi
}

get_project_dir() {
    local script_dir
    script_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
    # scripts/ -> u/ -> uspecs/ -> project root
    cd "$script_dir/../../.." && pwd
}

read_conf_param() {
    local param_name="$1"
    local conf_file
    conf_file="$(get_project_dir)/uspecs/u/conf.md"

    if [ ! -f "$conf_file" ]; then
        error "conf.md not found: $conf_file"
    fi

    local line raw
    line=$(grep -E "^- ${param_name}:" "$conf_file" | head -1 || true)
    raw="${line#*: }"

    if [ -z "$raw" ]; then
        error "Parameter '${param_name}' not found in conf.md"
    fi

    # trim leading/trailing whitespace and surrounding backticks
    local value
    value=$(echo "$raw" | sed 's/^[[:space:]`]*//' | sed 's/[[:space:]`]*$//')

    echo "$value"
}

cmd_change_new() {
    local change_name=""
    local issue_url=""
    local create_branch=""

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --issue-url)
                if [[ $# -lt 2 || -z "$2" ]]; then
                    error "--issue-url requires a URL argument"
                fi
                issue_url="$2"
                shift 2
                ;;
            --branch)
                create_branch="1"
                shift
                ;;
            *)
                if [ -z "$change_name" ]; then
                    change_name="$1"
                    shift
                else
                    error "Unknown argument: $1"
                fi
                ;;
        esac
    done

    if [ -z "$change_name" ]; then
        error "change-name is required"
    fi

    if [[ ! "$change_name" =~ ^[a-z0-9][a-z0-9-]*$ ]]; then
        error "change-name must be kebab-case (lowercase letters, numbers, hyphens): $change_name"
    fi

    local changes_folder_rel
    changes_folder_rel=$(read_conf_param "changes_folder")

    local project_dir
    project_dir=$(get_project_dir)

    local changes_folder="$project_dir/$changes_folder_rel"

    if [ ! -d "$changes_folder" ]; then
        error "Changes folder not found: $changes_folder"
    fi

    local timestamp
    timestamp=$(date -u +"%y%m%d%H%M")

    local folder_name="${timestamp}-${change_name}"
    local change_folder="$changes_folder/$folder_name"

    if [ -d "$change_folder" ]; then
        error "Change folder already exists: $change_folder"
    fi

    mkdir -p "$change_folder"

    local registered_at baseline
    registered_at=$(get_timestamp)
    baseline=$(get_baseline)

    local frontmatter="---"$'\n'
    frontmatter+="registered_at: $registered_at"$'\n'
    frontmatter+="change_id: $folder_name"$'\n'

    if [ -n "$baseline" ]; then
        frontmatter+="baseline: $baseline"$'\n'
    fi

    if [ -n "$issue_url" ]; then
        frontmatter+="issue_url: $issue_url"$'\n'
    fi

    frontmatter+="---"

    printf '%s\n' "$frontmatter" > "$change_folder/change.md"

    if [ -n "$create_branch" ]; then
        if git rev-parse --git-dir > /dev/null 2>&1; then
            if ! git checkout -b "$change_name" 2>/dev/null; then
                echo "Warning: Failed to create branch '$change_name'" >&2
            fi
        else
            echo "Warning: Not a git repository, cannot create branch" >&2
        fi
    fi

    echo "$changes_folder_rel/$folder_name"
}

convert_links_to_relative() {
    local folder="$1"

    if [ -z "$folder" ]; then
        error "folder path is required for convert_links_to_relative"
    fi

    if [ ! -d "$folder" ]; then
        error "Folder not found: $folder"
    fi

    # Find all .md files in the folder
    local md_files
    md_files=$(find "$folder" -maxdepth 1 -name "*.md" -type f)

    if [ -z "$md_files" ]; then
        # No markdown files to process, return success
        return 0
    fi

    # Process each markdown file
    while IFS= read -r file; do
        # Archive moves folder 2 levels deeper (changes/ -> changes/archive/yymm/)
        # Only paths starting with ../ need adjustment - add ../../ prefix
        #
        # Example: ](../foo) -> ](../../../foo)
        #
        # Skip (do not modify):
        # - http://, https:// (absolute URLs)
        # - # (anchors)
        # - / (absolute paths)
        # - ./ (current directory - stays in same folder)
        # - filename.ext (same folder files like impl.md, issue.md)

        # Add ../../ prefix to paths starting with ../
        # ](../ -> ](../../../
        if ! sed -i.bak -E 's#\]\(\.\./#](../../../#g' "$file"; then
            error "Failed to convert links in file: $file"
        fi
        rm -f "${file}.bak"
    done <<< "$md_files"

    return 0
}

cmd_change_archive() {
    local folder_name="$1"

    if [ -z "$folder_name" ]; then
        error "change-folder-name is required"
    fi

    local changes_folder_rel
    changes_folder_rel=$(read_conf_param "changes_folder")

    local project_dir
    project_dir=$(get_project_dir)

    local changes_folder="$project_dir/$changes_folder_rel"
    local path_to_change_folder="$changes_folder/$folder_name"

    if [ ! -d "$path_to_change_folder" ]; then
        error "Folder not found: $path_to_change_folder"
    fi

    local change_file="$path_to_change_folder/change.md"
    if [ ! -f "$change_file" ]; then
        error "change.md not found in folder: $path_to_change_folder"
    fi

    if [[ "$folder_name" == *archive* ]]; then
        error "Folder is already in archive: $folder_name"
    fi

    local uncompleted_count
    uncompleted_count=$(count_uncompleted_items "$path_to_change_folder")

    if [ "$uncompleted_count" -gt 0 ]; then
        echo "Cannot archive: $uncompleted_count uncompleted todo item(s) found"
        echo ""
        echo "Uncompleted items:"
        grep -rn "^\s*-\s*\[ \]" "$path_to_change_folder"/*.md 2>/dev/null | sed 's/^/  /'
        echo ""
        echo "Complete or cancel todo items before archiving"
        exit 1
    fi

    local timestamp
    timestamp=$(get_timestamp)

    # Insert archived_at into YAML front matter (before closing ---)
    local temp_file
    temp_file=$(mktemp)
    awk -v ts="$timestamp" '
        /^---$/ {
            if (count == 0) {
                print
                count++
            } else {
                print "archived_at: " ts
                print
            }
            next
        }
        { print }
    ' "$change_file" > "$temp_file"
    if mv "$temp_file" "$change_file"; then
        :  # Success, continue
    else
        rm -f "$temp_file"
        return 1
    fi

    # Add ../ prefix to relative links for archive folder depth
    if ! convert_links_to_relative "$path_to_change_folder"; then
        error "Failed to convert links to relative paths"
    fi

    local archive_folder="$changes_folder/archive"

    local date_prefix
    date_prefix=$(date -u +"%y%m%d%H%M")

    # Extract yymm for subfolder
    local yymm_prefix="${date_prefix:0:4}"

    local archive_subfolder="$archive_folder/$yymm_prefix"
    mkdir -p "$archive_subfolder"

    local change_name
    change_name=$(extract_change_name "$folder_name")

    local archive_path="$archive_subfolder/${date_prefix}-${change_name}"

    if [ -d "$archive_path" ]; then
        error "Archive folder already exists: $archive_path"
    fi

    if git rev-parse --git-dir > /dev/null 2>&1; then
        git add "$path_to_change_folder"
    fi

    move_folder "$path_to_change_folder" "$archive_path"

    if git rev-parse --git-dir > /dev/null 2>&1; then
        git add "$archive_path"
    fi

    echo "Archived change: $changes_folder_rel/archive/$yymm_prefix/${date_prefix}-${change_name}"
}

main() {
    if [ $# -lt 1 ]; then
        error "Usage: uspecs <command> [args...]"
    fi

    local command="$1"
    shift

    case "$command" in
        change)
            if [ $# -lt 1 ]; then
                error "Usage: uspecs change <subcommand> [args...]"
            fi
            local subcommand="$1"
            shift

            case "$subcommand" in
                new)
                    cmd_change_new "$@"
                    ;;
                archive)
                    cmd_change_archive "$@"
                    ;;
                *)
                    error "Unknown change subcommand: $subcommand"
                    ;;
            esac
            ;;
        *)
            error "Unknown command: $command"
            ;;
    esac
}

main "$@"
