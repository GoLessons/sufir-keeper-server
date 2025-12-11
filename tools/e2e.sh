#!/usr/bin/env bash

base_url="http://localhost:8080/api/v1"
compose_file="docker-compose.yml"

results=()
item_ids=()

function add_result() {
  local endpoint="$1"
  local method="$2"
  local status="$3"
  local result="$4"
  local notes="$5"
  results+=("$endpoint|$method|$status|$result|$notes")
}

function print_table() {
  printf "%-30s | %-6s | %-6s | %-5s | %s\n" "Endpoint" "Method" "Status" "Result" "Notes"
  for row in "${results[@]}"; do
    IFS="|" read -r endpoint method status result notes <<< "$row"
    printf "%-30s | %-6s | %-6s | %-5s | %s\n" "$endpoint" "$method" "$status" "$result" "$notes"
  done
}

function docker_compose_up() {
  docker compose -f "$compose_file" up -d
}

function wait_for_api_ready() {
  local attempts=20
  local delay_seconds=2
  local i=0
  while [ "$i" -lt "$attempts" ]; do
    local status
    status=$(curl -s -o /dev/null -w "%{http_code}" "$base_url/items")
    if [ "$status" = "401" ] || [ "$status" = "200" ]; then
      return 0
    fi
    sleep "$delay_seconds"
    i=$((i + 1))
  done
  return 1
}

function request_to_file() {
  local method="$1"
  local url="$2"
  local token="$3"
  local json_body="$4"
  local use_json="$5"
  local out_file="$6"
  local http_code
  if [ "$use_json" = "1" ]; then
    if [ -n "$token" ]; then
      http_code=$(curl -sS -X "$method" -H "Authorization: Bearer $token" -H "Content-Type: application/json" -d "$json_body" -o "$out_file" "$url" -w "%{http_code}")
    else
      http_code=$(curl -sS -X "$method" -H "Content-Type: application/json" -d "$json_body" -o "$out_file" "$url" -w "%{http_code}")
    fi
  else
    if [ -n "$token" ]; then
      http_code=$(curl -sS -X "$method" -H "Authorization: Bearer $token" -o "$out_file" "$url" -w "%{http_code}")
    else
      http_code=$(curl -sS -X "$method" -o "$out_file" "$url" -w "%{http_code}")
    fi
  fi
  printf "%s" "$http_code"
}

function request_multipart_file_to_file() {
  local url="$1"
  local token="$2"
  local file_id="$3"
  local file_path="$4"
  local out_file="$5"
  local http_code
  http_code=$(curl -sS -X POST -H "Authorization: Bearer $token" -H "X-File-ID: $file_id" -F "file=@$file_path" -o "$out_file" "$url" -w "%{http_code}")
  printf "%s" "$http_code"
}

function extract_json_value() {
  local body="$1"
  local key="$2"
  echo "$body" | sed -n "s/.*\"$key\"[[:space:]]*:[[:space:]]*\"\([^\"]*\)\".*/\1/p" | head -n 1
}

function generate_uuid() {
  cat /proc/sys/kernel/random/uuid
}

function compute_sha256() {
  local file_path="$1"
  sha256sum "$file_path" | awk '{print $1}'
}

function main() {
  docker_compose_up || true
  wait_for_api_ready || {
    add_result "/items" "GET" "" "FAIL" "API не готов"
    print_table
    exit 1
  }

  local unique_login
  unique_login="e2e_user_$(date +%s)"
  local user_password
  user_password="strongpassword123"

  local http_code body tmp
  tmp=$(mktemp)
  http_code=$(request_to_file "POST" "$base_url/register" "" "{\"login\":\"$unique_login\",\"password\":\"$user_password\"}" "1" "$tmp")
  body=$(cat "$tmp")
  if [ "$http_code" = "201" ] || [ "$http_code" = "409" ]; then
    add_result "/register" "POST" "$http_code" "PASS" "Регистрация"
  else
    add_result "/register" "POST" "$http_code" "FAIL" "Регистрация"
  fi

  tmp=$(mktemp)
  http_code=$(request_to_file "POST" "$base_url/auth" "" "{\"login\":\"$unique_login\",\"password\":\"$user_password\"}" "1" "$tmp")
  body=$(cat "$tmp")
  local access_token refresh_token
  access_token=$(extract_json_value "$body" "access_token")
  refresh_token=$(extract_json_value "$body" "refresh_token")
  if [ "$http_code" = "200" ] && [ -n "$access_token" ] && [ -n "$refresh_token" ]; then
    add_result "/auth" "POST" "$http_code" "PASS" "Аутентификация"
  else
    add_result "/auth" "POST" "$http_code" "FAIL" "Аутентификация"
  fi

  tmp=$(mktemp)
  http_code=$(request_to_file "PATCH" "$base_url/auth" "" "{\"refresh_token\":\"$refresh_token\"}" "1" "$tmp")
  body=$(cat "$tmp")
  local new_access_token new_refresh_token
  new_access_token=$(extract_json_value "$body" "access_token")
  new_refresh_token=$(extract_json_value "$body" "refresh_token")
  if [ "$http_code" = "200" ] && [ -n "$new_access_token" ] && [ -n "$new_refresh_token" ]; then
    access_token="$new_access_token"
    refresh_token="$new_refresh_token"
    add_result "/auth" "PATCH" "$http_code" "PASS" "Обновление токена"
  else
    add_result "/auth" "PATCH" "$http_code" "FAIL" "Обновление токена"
  fi

  tmp=$(mktemp)
  http_code=$(request_to_file "POST" "$base_url/auth-verify" "$access_token" "" "0" "$tmp")
  if [ "$http_code" = "204" ]; then
    add_result "/auth-verify" "POST" "$http_code" "PASS" "Готовность авторизации"
  else
    add_result "/auth-verify" "POST" "$http_code" "FAIL" "Готовность авторизации"
  fi

  local file_id file_name file_path file_sha256
  file_id=$(generate_uuid)
  file_name="e2e_upload.txt"
  file_path=$(mktemp)
  echo "e2e content" > "$file_path"
  file_sha256=$(compute_sha256 "$file_path")

  local item_credential_json item_card_json item_text_json item_binary_json
  item_credential_json="{\"title\":\"Запись CREDENTIAL\",\"data\":{\"type\":\"CREDENTIAL\",\"login\":\"user@example.com\",\"password\":\"password123\"}}"
  item_card_json="{\"title\":\"Запись CARD\",\"data\":{\"type\":\"CARD\",\"card_number\":\"4111111111111111\",\"card_holder\":\"IVAN IVANOV\",\"expiry_date\":\"12/25\",\"cvv\":\"123\"}}"
  item_text_json="{\"title\":\"Запись TEXT\",\"data\":{\"type\":\"TEXT\",\"value\":\"Мой секретный текст\"}}"
  item_binary_json="{\"title\":\"Запись BINARY\",\"data\":{\"type\":\"BINARY\",\"filename\":\"$file_name\",\"id\":\"$file_id\"}}"

  tmp=$(mktemp)
  http_code=$(request_to_file "POST" "$base_url/items" "$access_token" "$item_credential_json" "1" "$tmp")
  body=$(cat "$tmp")
  if [ "$http_code" = "201" ]; then
    add_result "/items" "POST" "$http_code" "PASS" "Создание CREDENTIAL"
    item_ids+=("$(extract_json_value "$body" "id")")
  else
    add_result "/items" "POST" "$http_code" "FAIL" "Создание CREDENTIAL"
  fi

  tmp=$(mktemp)
  http_code=$(request_to_file "POST" "$base_url/items" "$access_token" "$item_card_json" "1" "$tmp")
  body=$(cat "$tmp")
  if [ "$http_code" = "201" ]; then
    add_result "/items" "POST" "$http_code" "PASS" "Создание CARD"
    item_ids+=("$(extract_json_value "$body" "id")")
  else
    add_result "/items" "POST" "$http_code" "FAIL" "Создание CARD"
  fi

  tmp=$(mktemp)
  http_code=$(request_to_file "POST" "$base_url/items" "$access_token" "$item_text_json" "1" "$tmp")
  body=$(cat "$tmp")
  if [ "$http_code" = "201" ]; then
    add_result "/items" "POST" "$http_code" "PASS" "Создание TEXT"
    item_ids+=("$(extract_json_value "$body" "id")")
  else
    add_result "/items" "POST" "$http_code" "FAIL" "Создание TEXT"
  fi

  tmp=$(mktemp)
  http_code=$(request_to_file "POST" "$base_url/items" "$access_token" "$item_binary_json" "1" "$tmp")
  body=$(cat "$tmp")
  if [ "$http_code" = "201" ]; then
    add_result "/items" "POST" "$http_code" "PASS" "Создание BINARY"
    item_ids+=("$(extract_json_value "$body" "id")")
  else
    add_result "/items" "POST" "$http_code" "FAIL" "Создание BINARY"
  fi

  for item_id in "${item_ids[@]}"; do
    tmp=$(mktemp)
    http_code=$(request_to_file "GET" "$base_url/items/$item_id" "$access_token" "" "0" "$tmp")
    body=$(cat "$tmp")
    if [ "$http_code" = "200" ]; then
      add_result "/items/{id}" "GET" "$http_code" "PASS" "$item_id"
    else
      add_result "/items/{id}" "GET" "$http_code" "FAIL" "$item_id"
    fi
  done

  if [ "${#item_ids[@]}" -gt 0 ]; then
    local first_id
    first_id="${item_ids[0]}"
    tmp=$(mktemp)
    http_code=$(request_to_file "PUT" "$base_url/items/$first_id" "$access_token" "{\"title\":\"Обновленный заголовок\"}" "1" "$tmp")
    body=$(cat "$tmp")
    if [ "$http_code" = "200" ]; then
      add_result "/items/{id}" "PUT" "$http_code" "PASS" "$first_id"
      tmp=$(mktemp)
      http_code=$(request_to_file "GET" "$base_url/items/$first_id" "$access_token" "" "0" "$tmp")
      body=$(cat "$tmp")
      if [ "$http_code" = "200" ]; then
        add_result "/items/{id}" "GET" "$http_code" "PASS" "После обновления"
      else
        add_result "/items/{id}" "GET" "$http_code" "FAIL" "После обновления"
      fi
    else
      add_result "/items/{id}" "PUT" "$http_code" "FAIL" "$first_id"
    fi
  fi

  tmp=$(mktemp)
  http_code=$(request_to_file "GET" "$base_url/items" "$access_token" "" "0" "$tmp")
  body=$(cat "$tmp")
  if [ "$http_code" = "200" ]; then
    add_result "/items" "GET" "$http_code" "PASS" "Список"
  else
    add_result "/items" "GET" "$http_code" "FAIL" "Список"
  fi

  tmp=$(mktemp)
  http_code=$(request_to_file "GET" "$base_url/items?type=CREDENTIAL" "$access_token" "" "0" "$tmp")
  body=$(cat "$tmp")
  if [ "$http_code" = "200" ]; then
    add_result "/items?type=CREDENTIAL" "GET" "$http_code" "PASS" "Фильтр по типу"
  else
    add_result "/items?type=CREDENTIAL" "GET" "$http_code" "FAIL" "Фильтр по типу"
  fi

  tmp=$(mktemp)
  http_code=$(request_to_file "GET" "$base_url/items?s=TEXT" "$access_token" "" "0" "$tmp")
  body=$(cat "$tmp")
  if [ "$http_code" = "200" ]; then
    add_result "/items?s=TEXT" "GET" "$http_code" "PASS" "Поиск"
  else
    add_result "/items?s=TEXT" "GET" "$http_code" "FAIL" "Поиск"
  fi

  tmp=$(mktemp)
  http_code=$(request_to_file "GET" "$base_url/items?limit=2&offset=0" "$access_token" "" "0" "$tmp")
  body=$(cat "$tmp")
  if [ "$http_code" = "200" ]; then
    add_result "/items?limit=2&offset=0" "GET" "$http_code" "PASS" "Пагинация"
  else
    add_result "/items?limit=2&offset=0" "GET" "$http_code" "FAIL" "Пагинация"
  fi

  if [ "${#item_ids[@]}" -gt 0 ]; then
    local delete_id
    delete_id="${item_ids[${#item_ids[@]}-1]}"
    tmp=$(mktemp)
    http_code=$(request_to_file "DELETE" "$base_url/items/$delete_id" "$access_token" "" "0" "$tmp")
    body=$(cat "$tmp")
    if [ "$http_code" = "204" ]; then
      add_result "/items/{id}" "DELETE" "$http_code" "PASS" "$delete_id"
      tmp=$(mktemp)
      http_code=$(request_to_file "GET" "$base_url/items/$delete_id" "$access_token" "" "0" "$tmp")
      body=$(cat "$tmp")
      if [ "$http_code" = "404" ]; then
        add_result "/items/{id}" "GET" "$http_code" "PASS" "После удаления"
      else
        add_result "/items/{id}" "GET" "$http_code" "FAIL" "После удаления"
      fi
    else
      add_result "/items/{id}" "DELETE" "$http_code" "FAIL" "$delete_id"
    fi
  fi

  file_id=$(generate_uuid)

  tmp=$(mktemp)
  http_code=$(request_to_file "POST" "$base_url/files/presign" "$access_token" "{\"fileId\":\"$file_id\",\"filename\":\"$file_name\",\"mime\":\"text/plain\",\"checksum\":\"$file_sha256\"}" "1" "$tmp")
  body=$(cat "$tmp")
  local presign_upload_url presign_key
  presign_upload_url=$(extract_json_value "$body" "upload_url")
  presign_key=$(extract_json_value "$body" "key")
  if [ "$http_code" = "200" ] && [ -n "$presign_upload_url" ] && [ -n "$presign_key" ]; then
    add_result "/files/presign" "POST" "$http_code" "PASS" "Presign"
  else
    add_result "/files/presign" "POST" "$http_code" "FAIL" "Presign"
  fi

  if [ -n "$presign_upload_url" ]; then
    local upload_url_full
    if echo "$presign_upload_url" | grep -qE '^https?://'; then
      upload_url_full="$presign_upload_url"
    else
      upload_url_full="http://localhost:8080$presign_upload_url"
    fi
    local form_fields_block
    form_fields_block=$(echo "$body" | sed -n 's/.*\"form_fields\"[[:space:]]*:[[:space:]]*{\([^}]*\)}.*/\1/p')
    local fields_present_key=0
    local -a args
    args=( -sS -X POST -H "Authorization: Bearer $access_token" )
    IFS=$'\n' read -r -d '' -a kvs < <(echo "$form_fields_block" | grep -o '\"[^\"[:space:]]\+\"[[:space:]]*:[[:space:]]*\"[^\"]*\"' && printf '\0')
    for kv in "${kvs[@]}"; do
      local k v
      k=$(echo "$kv" | sed -n 's/^\"\([^\"]\+\)\"[[:space:]]*:[[:space:]]*\".*\"$/\1/p')
      v=$(echo "$kv" | sed -n 's/^\"[^\\"]\+\"[[:space:]]*:[[:space:]]*\"\(.*\)\"$/\1/p')
      if [ -n "$k" ]; then
        if [ "$k" = "key" ]; then fields_present_key=1; fi
        if [ "$k" = "success_action_status" ]; then continue; fi
        args+=( --form-string "$k=$v" )
      fi
    done
    if [ "$fields_present_key" -eq 0 ] && [ -n "$presign_key" ]; then
      args+=( --form-string "key=$presign_key" )
    fi
    args+=( -F "file=@$file_path" "$upload_url_full" -w "%{http_code}" -o /dev/null )
    
    local upload_code
    upload_code=$(curl "${args[@]}")
    if [ "$upload_code" = "204" ]; then
      add_result "/files" "POST" "$upload_code" "PASS" "Загрузка файла"
      
      # Wait for webhook to process (retry loop)
      local attempts=0
      while [ $attempts -lt 10 ]; do
        if curl -sS -H "Authorization: Bearer $access_token" "$base_url/items/$file_id" -f >/dev/null 2>&1; then
           break
        fi
        sleep 1
        attempts=$((attempts + 1))
      done

    else
      add_result "/files" "POST" "$upload_code" "FAIL" "Загрузка файла"
    fi
  else
    add_result "/files" "POST" "" "FAIL" "Загрузка файла (нет presign)"
  fi

  local downloaded_path
  downloaded_path=$(mktemp)
  local download_out
  download_out=$(curl -sS -H "Authorization: Bearer $access_token" "$base_url/files/$file_id" -o "$downloaded_path" -w "%{http_code}")
  local download_code
  download_code="$download_out"
  if [ "$download_code" = "200" ]; then
    local downloaded_sha256
    downloaded_sha256=$(compute_sha256 "$downloaded_path")
    if [ "$downloaded_sha256" = "$file_sha256" ]; then
      add_result "/files/{fileId}" "GET" "$download_code" "PASS" "Скачивание"
    else
      add_result "/files/{fileId}" "GET" "$download_code" "FAIL" "Хеш не совпадает"
    fi
  else
    add_result "/files/{fileId}" "GET" "$download_code" "FAIL" "Скачивание"
  fi

  tmp=$(mktemp)
  http_code=$(request_to_file "DELETE" "$base_url/auth" "$access_token" "" "0" "$tmp")
  body=$(cat "$tmp")
  if [ "$http_code" = "204" ]; then
    add_result "/auth" "DELETE" "$http_code" "PASS" "Выход"
  else
    add_result "/auth" "DELETE" "$http_code" "FAIL" "Выход"
  fi

  print_table
}

main
