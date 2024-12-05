# Markdown to HTML 转换器

一个功能强大的 Markdown 转 HTML 工具，支持代码高亮、表格对齐和嵌套元素。

## 特性

- **代码块支持**
  - 语法高亮（支持多种编程语言）
  - 保留原始格式（空格、缩进）
  - 美观的代码显示效果

- **表格增强**
  - 支持左中右对齐
  - 自动处理表格边框
  - 响应式表格布局

- **引用增强**
  - 支持嵌套代码块
  - 支持嵌套表格
  - 保留原始格式

- **MinIO 集成**
  - 自动上传生成的 HTML
  - 生成可访问的 URL
  - 支持自定义存储配置

## 安装

1. 确保已安装 Go 1.16 或更高版本
2. 克隆仓库：
   ```bash
   git clone [repository-url]
   cd markdown2html
   ```
3. 安装依赖：
   ```bash
   go mod download
   ```

## 配置

创建 `config.json` 文件：

```json
{
  "s3": {
    "endpoint": "your-minio-endpoint",
    "port": 9000,
    "bucket": "your-bucket",
    "accessKey": "your-access-key",
    "secretKey": "your-secret-key",
    "region": "us-east-1",
    "useSSL": true
  },
  "custom": {
    "baseUrl": "your-base-url",
    "path": "pages",
    "outputDir": "output"
  }
}
```

## 使用方法

1. 基本使用：
   ```bash
   ./markdown2html input.md
   ```

2. 指定配置文件：
   ```bash
   ./markdown2html -c custom_config.json input.md
   ```

3. 批量转换：
   ```bash
   ./markdown2html file1.md file2.md file3.md
   ```

## 示例

### 代码块示例

```python
def hello_world():
    print("Hello, World!")
    
    # 带缩进的注释
    for i in range(3):
        print(f"Count: {i}")
```

### 表格示例

| 功能 | 支持情况 | 备注 |
|:---|:---:|---:|
| 代码块 | | 支持语法高亮 |
| 引用 | | 支持嵌套元素 |
| 表格 | | 支持对齐 |

### 引用示例

> 这是一个引用示例
> 
> 可以包含代码块：
> ```go
> func main() {
>     fmt.Println("Hello!")
> }
> ```
> 
> 也可以包含表格：
> | 名称 | 描述 |
> |:---|:---|
> | 项目1 | 测试项目 |


## 贡献

欢迎提交 Issue 和 Pull Request！

## 许可证

MIT License
