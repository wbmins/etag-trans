package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
)

// 定义 db.text.json 的结构部分，只取必要字段
type DB struct {
	Data []struct {
		FrontMatters struct {
			Key string `json:"key"`
		} `json:"frontMatters"`
		Data map[string]struct {
			Name string `json:"name"`
		} `json:"data"`
	} `json:"data"`
}

func main() {
	log.Println("🚀 开始更新 tagMap")

	// ------------------------
	// 1️⃣ 下载最新 db.text.json
	// ------------------------
	url := "https://raw.githubusercontent.com/EhTagTranslation/Database/refs/heads/release/db.text.json"
	log.Println("📥 下载 db.text.json 中:", url)
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal("❌ 下载 db.text.json 失败:", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Fatal("❌ 下载 db.text.json 返回状态码:", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("❌ 读取响应体失败:", err)
	}
	log.Println("✅ 下载并读取 db.text.json 成功")

	// ------------------------
	// 2️⃣ 解析 JSON
	// ------------------------
	var db DB
	if err := json.Unmarshal(body, &db); err != nil {
		log.Fatal("❌ 解析 db.text.json 失败:", err)
	}
	log.Printf("✅ JSON 解析成功，发现 %d 分类\n", len(db.Data))

	// ------------------------
	// 3️⃣ 自动遍历所有 frontMatters.key
	// ------------------------
	output := make(map[string]map[string]string)
	for _, d := range db.Data {
		key := d.FrontMatters.Key
		if _, exists := output[key]; !exists {
			subMap := make(map[string]string)
			for k, v := range d.Data {
				if v.Name != "" {
					subMap[k] = v.Name
				} else {
					subMap[k] = k
				}
			}
			output[key] = subMap
			log.Printf("🔹 %q: %d 条标签\n", key, len(subMap))
		}
	}

	// ------------------------
	// 4️⃣ 生成 Go tagMap 字符串
	// ------------------------
	goMapStr := ""
	for category, subMap := range output {
		goMapStr += fmt.Sprintf("\t%q: {\n", category)
		for en, zh := range subMap {
			goMapStr += fmt.Sprintf("\t\t%q: %q,\n", en, zh)
		}
		goMapStr += "\t},\n"
	}
	goMapStr += "}\n"

	// ------------------------
	// 5️⃣ 读取 query.go 并替换 tagMap
	// ------------------------
	queryPath := "api/query.go"
	queryData, err := os.ReadFile(queryPath)
	if err != nil {
		log.Fatal("❌ 读取 query.go 失败:", err)
	}
	querySrc := string(queryData)

	// 找到 var tagMap = map[string]map[string]string{ 这一行
	reLine := regexp.MustCompile(`(?m)^var\s+tagMap\s*=\s*map\[string\]map\[string\]string\s*{`)
	loc := reLine.FindStringIndex(querySrc)
	if loc == nil {
		log.Fatal("❌ query.go 中没有找到 var tagMap = map[string]map[string]string{ 这一行")
	}

	// prefix 包含这一行，覆盖旧内容
	prefix := querySrc[:loc[1]]
	newQuerySrc := prefix + "\n" + goMapStr

	// 写回文件
	if err := os.WriteFile(queryPath, []byte(newQuerySrc), 0644); err != nil {
		log.Fatal("❌ 写入 query.go 失败:", err)
	}

	log.Println("✅ 已更新 api/query.go 中的 tagMap")
	log.Println("🎉 更新完成！")
}
