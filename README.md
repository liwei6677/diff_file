# diff_file

一个轻量级的文本 / JSON 对比工具，提供浏览器界面和 HTTP API 两种使用方式。

## 🌐 在线访问（GitHub Pages）

无需安装任何依赖，直接在浏览器中使用：

👉 **[https://liwei6677.github.io/diff_file/](https://liwei6677.github.io/diff_file/)**

> 页面基于纯前端 JavaScript 实现，所有对比逻辑均在浏览器本地执行，无需服务端。

## 功能

- **文本对比**（逐行）：基于 LCS 算法，支持字符级内联差异展示
- **JSON 对比**（Key-Value）：将嵌套 JSON 展开为点号路径后逐 key 比较
- **浏览器界面**：访问在线地址或本地服务即可使用可视化对比工具
- **HTTP API**：通过 REST 接口以编程方式调用对比功能（需本地运行 Go 服务）

## 部署

### 静态部署（GitHub Pages）

仓库已配置 GitHub Actions 工作流（`.github/workflows/pages.yml`），每次推送到 `main` 分支时自动部署到 GitHub Pages。

**首次启用步骤：**
1. 打开仓库 **Settings → Pages**。
2. 在 **Build and deployment → Source** 下选择 **GitHub Actions**。
3. 等待约 1-2 分钟，页面即可通过 `https://liwei6677.github.io/diff_file/` 访问。

### 本地运行（含 Go 后端）

```bash
go run .
# 服务默认监听 :8080
```

打开浏览器访问 [http://localhost:8080](http://localhost:8080) 使用可视化界面。

## API 文档

所有接口均返回 `application/json`，并附带 `Access-Control-Allow-Origin: *` 跨域头。

---

### `GET /health`

健康检查。

**响应示例**

```json
{"status": "ok"}
```

---

### `POST /api/diff/text`

对两段文本进行逐行对比。

**请求体**

```json
{
  "left":  "原始内容（每行用 \\n 分隔）",
  "right": "对比内容（每行用 \\n 分隔）"
}
```

**响应体** — `TextDiffResult`

```json
{
  "diffs": [
    {
      "type":       "equal",
      "value":      "hello",
      "left_line":  1,
      "right_line": 1
    },
    {
      "type":        "changed",
      "left_value":  "world",
      "right_value": "golang",
      "left_line":   2,
      "right_line":  2
    }
  ],
  "summary": {
    "added":   0,
    "removed": 0,
    "changed": 1,
    "total":   1
  }
}
```

`type` 取值：

| 值        | 含义                      | 有效字段                                                       |
|-----------|---------------------------|----------------------------------------------------------------|
| `equal`   | 两侧相同                  | `value`, `left_line`, `right_line`                             |
| `added`   | 仅右侧存在                | `value`, `right_line`                                          |
| `removed` | 仅左侧存在                | `value`, `left_line`                                           |
| `changed` | 连续的删除+新增，视为修改 | `left_value`, `right_value`, `left_line`, `right_line`         |

> 输入行数上限为 5 000 行；超出时返回 HTTP 400。

**curl 示例**

```bash
curl -X POST http://localhost:8080/api/diff/text \
  -H 'Content-Type: application/json' \
  -d '{"left":"hello\nworld","right":"hello\ngolang"}'
```

---

### `POST /api/diff/json`

对两个 JSON 字符串进行 Key-Value 对比（嵌套结构展开为点号路径，数组使用 `[n]` 下标）。

**请求体**

```json
{
  "left":  "{\"name\":\"Alice\",\"age\":30}",
  "right": "{\"name\":\"Bob\",\"age\":30,\"city\":\"NYC\"}"
}
```

**响应体** — `JSONDiffResult`

```json
{
  "diffs": [
    {
      "path":        "city",
      "type":        "added",
      "right_value": "NYC"
    },
    {
      "path":        "name",
      "type":        "changed",
      "left_value":  "Alice",
      "right_value": "Bob"
    }
  ],
  "summary": {
    "added":   1,
    "removed": 0,
    "changed": 1,
    "total":   2
  }
}
```

`type` 取值：

| 值        | 含义           | 有效字段                    |
|-----------|----------------|-----------------------------|
| `added`   | 仅右侧存在该键 | `right_value`               |
| `removed` | 仅左侧存在该键 | `left_value`                |
| `changed` | 两侧值不同     | `left_value`, `right_value` |

> `left_value` / `right_value` 为原始 JSON 值（字符串、数字、布尔、null、对象、数组均可）。

**curl 示例**

```bash
curl -X POST http://localhost:8080/api/diff/json \
  -H 'Content-Type: application/json' \
  -d '{"left":"{\"name\":\"Alice\",\"age\":30}","right":"{\"name\":\"Bob\",\"age\":30,\"city\":\"NYC\"}"}'
```

## 错误响应

所有接口在出错时返回对应 HTTP 状态码及 JSON 错误体：

```json
{"error": "错误描述"}
```

| 情况                  | 状态码 |
|-----------------------|--------|
| 请求体无法解析为 JSON | 400    |
| 内部 JSON 解析失败    | 400    |
| 输入超过行数限制      | 400    |
