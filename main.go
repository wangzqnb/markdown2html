package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"html"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/russross/blackfriday/v2"
)

type Config struct {
	S3 struct {
		Endpoint  string `json:"endpoint"`
		Port      int    `json:"port"`
		Bucket    string `json:"bucket"`
		AccessKey string `json:"accessKey"`
		SecretKey string `json:"secretKey"`
		Region    string `json:"region"`
		UseSSL    bool   `json:"useSSL"`
	} `json:"s3"`
	Custom struct {
		BaseURL   string `json:"baseUrl"`
		Path      string `json:"path"`
		OutputDir string `json:"outputDir"`
	} `json:"custom"`
}

func loadConfig(configPath string) (*Config, error) {
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %v", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}

	return &config, nil
}

// 获取程序所在目录的配置文件路径
func getDefaultConfigPath() string {
	// 获取程序可执行文件的路径
	execPath, err := os.Executable()
	if err != nil {
		return "config.json" // 如果获取失败，返回当前目录
	}

	// 获取程序所在目录
	execDir := filepath.Dir(execPath)

	// 拼接配置文件路径
	return filepath.Join(execDir, "config.json")
}

func uploadToS3(config *Config, filePath string, content []byte) (string, error) {
	fmt.Printf("正在连接 MinIO 服务器...\n")

	// 创建自定义解析器
	customResolver := aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
		if service == s3.ServiceID {
			return aws.Endpoint{
				URL:           fmt.Sprintf("http://%s:%d", strings.TrimPrefix(strings.TrimPrefix(config.S3.Endpoint, "http://"), "https://"), config.S3.Port),
				SigningRegion: config.S3.Region,
			}, nil
		}
		return aws.Endpoint{}, fmt.Errorf("unknown service")
	})

	// 创建S3客户端配置
	cfg := aws.Config{
		Region: config.S3.Region,
		Credentials: credentials.NewStaticCredentialsProvider(
			config.S3.AccessKey,
			config.S3.SecretKey,
			"",
		),
		EndpointResolver: customResolver,
	}

	// 创建S3客户端
	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	// 生成文件名（使用原始文件名）
	fileName := filepath.Base(filePath)

	// 生成年/月路径 (YYYY/MM)
	currentTime := time.Now()
	datePath := currentTime.Format("2006/01")

	// 组合最终的文件路径
	s3Key := fmt.Sprintf("%s/%s", datePath, fileName)

	// 构建上传请求
	input := &s3.PutObjectInput{
		Bucket:      aws.String(config.S3.Bucket),
		Key:         aws.String(s3Key),
		Body:        bytes.NewReader(content),
		ContentType: aws.String("text/html"),
	}

	// 上传文件
	_, err := s3Client.PutObject(context.TODO(), input)
	if err != nil {
		return "", fmt.Errorf("上传失败: %v", err)
	}

	// 生成自定义URL
	customURL := fmt.Sprintf("%s%s/%s",
		strings.TrimRight(config.Custom.BaseURL, "/"),
		strings.TrimRight(config.Custom.Path, "/"),
		s3Key)

	return customURL, nil
}

// 判断是否为分隔符行
func isAlignmentRow(cell string) bool {
	// 去除首尾空格和可能存在的冒号
	trimmed := strings.TrimSpace(strings.Trim(cell, ":"))
	if trimmed == "" {
		return false
	}
	// 检查是否只包含 - 字符
	for _, ch := range trimmed {
		if ch != '-' {
			return false
		}
	}
	return true
}

// 获取对齐方式
func getAlignment(cell string) string {
	cell = strings.TrimSpace(cell)
	hasLeft := strings.HasPrefix(cell, ":")
	hasRight := strings.HasSuffix(strings.TrimRight(cell, " -"), ":")

	if hasLeft && hasRight {
		return "center"
	} else if hasLeft {
		return "left"
	} else if hasRight {
		return "right"
	}
	return "left" // 默认左对齐
}

// 处理表格行，支持省略首尾 |
func processTableRow(line string) []string {
	// 去除首尾空格
	line = strings.TrimSpace(line)

	// 如果行首尾有 |，去除它们
	if strings.HasPrefix(line, "|") {
		line = line[1:]
	}
	if strings.HasSuffix(line, "|") {
		line = line[:len(line)-1]
	}

	// 分割单元格并处理每个单元格
	cells := strings.Split(line, "|")
	for i := range cells {
		cells[i] = strings.TrimSpace(cells[i])
	}

	return cells
}

// 判断是否为有效的表格行
func isValidTableRow(line string) bool {
	// 去除首尾空格
	line = strings.TrimSpace(line)

	// 检查是否包含至少一个 |
	if !strings.Contains(line, "|") {
		return false
	}

	// 分割单元格
	cells := processTableRow(line)

	// 检查是否至少有一个非空单元格
	hasNonEmpty := false
	for _, cell := range cells {
		if strings.TrimSpace(cell) != "" {
			hasNonEmpty = true
			break
		}
	}

	return hasNonEmpty
}

// 判断是否为有效的分隔行
func isValidSeparatorRow(line string) bool {
	// 去除首尾空格
	line = strings.TrimSpace(line)

	// 检查是否包含至少一个 | 和 -
	if !strings.Contains(line, "|") || !strings.Contains(line, "-") {
		return false
	}

	// 分割单元格
	cells := processTableRow(line)

	// 检查每个单元格是否都是有效的分隔符
	for _, cell := range cells {
		cell = strings.TrimSpace(cell)
		if cell == "" {
			continue
		}
		if !isAlignmentRow(cell) {
			return false
		}
	}

	return true
}

func escapeHTML(s string) string {
	return html.EscapeString(s)
}

// 处理代码块，保留原始格式
func processCodeBlock(lines []string, startLine int) (string, int) {
	// 获取语言标记
	lang := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(lines[startLine]), "```"))

	var codeLines []string
	i := startLine + 1

	// 收集代码行直到遇到结束标记
	for i < len(lines) {
		line := lines[i]
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			break
		}
		codeLines = append(codeLines, line) // 使用原始行，保留空格和缩进
		i++
	}

	// 连接代码行，保留原始格式
	code := strings.Join(codeLines, "\n")

	// 使用 html.EscapeString 保留换行符和空格
	code = html.EscapeString(code)

	// 构建HTML
	var langClass string
	if lang != "" {
		langClass = fmt.Sprintf(` class="language-%s"`, lang)
	}
	html := fmt.Sprintf("<pre><code%s>%s</code></pre>", langClass, code)

	return html, i
}

// 处理引用内容，支持嵌套的代码块和表格
func processQuoteContent(lines []string, startLine int) (string, int) {
	var quoteLines []string
	i := startLine

	// 收集引用行直到遇到非引用行
	for i < len(lines) {
		line := lines[i]
		if !strings.HasPrefix(strings.TrimSpace(line), ">") {
			break
		}

		// 移除引用标记，但保留其他空格
		quoteLine := strings.TrimPrefix(line, ">")
		if strings.HasPrefix(quoteLine, " ") {
			quoteLine = quoteLine[1:]
		}

		// 检查是否是代码块开始
		if strings.HasPrefix(strings.TrimSpace(quoteLine), "```") {
			// 收集代码块内容
			var codeLines []string
			lang := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(quoteLine), "```"))
			i++

			for i < len(lines) {
				line = lines[i]
				if !strings.HasPrefix(strings.TrimSpace(line), ">") {
					break
				}
				quoteLine = strings.TrimPrefix(line, ">")
				if strings.HasPrefix(quoteLine, " ") {
					quoteLine = quoteLine[1:]
				}

				if strings.HasPrefix(strings.TrimSpace(quoteLine), "```") {
					break
				}
				codeLines = append(codeLines, quoteLine)
				i++
			}

			// 处理代码块
			code := strings.Join(codeLines, "\n")
			code = html.EscapeString(code)
			var langClass string
			if lang != "" {
				langClass = fmt.Sprintf(` class="language-%s"`, lang)
			}
			quoteLines = append(quoteLines, fmt.Sprintf("<pre><code%s>%s</code></pre>", langClass, code))

			if i < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i]), ">") {
				i++ // 跳过结束标记
			}
		} else {
			quoteLines = append(quoteLines, quoteLine)
			i++
		}
	}

	// 处理引用内容中的其他Markdown语法
	quotedContent := strings.Join(quoteLines, "\n")
	extensions := blackfriday.CommonExtensions
	renderer := blackfriday.NewHTMLRenderer(blackfriday.HTMLRendererParameters{
		Flags: blackfriday.CommonHTMLFlags | blackfriday.UseXHTML,
	})
	innerHtml := blackfriday.Run([]byte(quotedContent), blackfriday.WithExtensions(extensions), blackfriday.WithRenderer(renderer))

	return fmt.Sprintf("<blockquote>\n%s\n</blockquote>", string(innerHtml)), i - 1
}

func convertMarkdownToHTML(content string) string {
	lines := strings.Split(content, "\n")
	var htmlLines []string
	inTable := false
	inCodeBlock := false
	var tableRows []string
	var alignments []string
	var headerCellCount int

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// 处理代码块
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			if !inCodeBlock {
				// 代码块开始
				html, endLine := processCodeBlock(lines, i)
				htmlLines = append(htmlLines, html)
				i = endLine
				continue
			}
		}

		// 检查是否为表格行
		isTableRow := isValidTableRow(line)
		isSeparatorRow := isValidSeparatorRow(line)

		if isTableRow || isSeparatorRow {
			if !inTable {
				// 如果这是第一行，检查下一行是否为分隔行
				if i+1 < len(lines) && isValidSeparatorRow(lines[i+1]) {
					inTable = true
					tableRows = []string{}
					alignments = nil
					cells := processTableRow(line)
					headerCellCount = len(cells)
				} else if isSeparatorRow {
					// 如果第一行就是分隔行，跳过
					continue
				} else {
					// 不是有效的表格开始
					htmlLines = append(htmlLines, line)
					continue
				}
			}

			cells := processTableRow(line)

			// 检查单元格数量是否与表头一致（允许有空单元格）
			if inTable && !isSeparatorRow && len(cells) != headerCellCount {
				// 如果单元格数量不匹配，结束当前表格
				if len(tableRows) > 0 {
					table := "<table>\n<thead>\n" + tableRows[0] + "\n</thead>\n<tbody>\n"
					table += strings.Join(tableRows[1:], "\n")
					table += "\n</tbody>\n</table>"
					htmlLines = append(htmlLines, table)
				}
				inTable = false
				htmlLines = append(htmlLines, line)
				continue
			}

			if isSeparatorRow {
				alignments = make([]string, len(cells))
				for j, cell := range cells {
					alignments[j] = getAlignment(cell)
				}
				continue
			}

			row := "<tr>"
			for j, cell := range cells {
				align := ""
				if len(alignments) > j {
					align = fmt.Sprintf(` align="%s"`, alignments[j])
				}
				if len(tableRows) == 0 {
					row += fmt.Sprintf("<th%s>%s</th>", align, escapeHTML(cell))
				} else {
					row += fmt.Sprintf("<td%s>%s</td>", align, escapeHTML(cell))
				}
			}
			row += "</tr>"
			tableRows = append(tableRows, row)

			// 检查是否为表格结束
			if i+1 >= len(lines) || (!isValidTableRow(lines[i+1]) && !isValidSeparatorRow(lines[i+1])) {
				table := "<table>\n<thead>\n" + tableRows[0] + "\n</thead>\n<tbody>\n"
				table += strings.Join(tableRows[1:], "\n")
				table += "\n</tbody>\n</table>"
				htmlLines = append(htmlLines, table)
				inTable = false
			}
		} else {
			if inTable {
				table := "<table>\n<thead>\n" + tableRows[0] + "\n</thead>\n<tbody>\n"
				table += strings.Join(tableRows[1:], "\n")
				table += "\n</tbody>\n</table>"
				htmlLines = append(htmlLines, table)
				inTable = false
			}

			// 处理引用语法
			if strings.HasPrefix(line, ">") {
				quotedHtml, endLine := processQuoteContent(lines, i)
				htmlLines = append(htmlLines, quotedHtml)
				i = endLine
			} else if line != "" {
				extensions := blackfriday.CommonExtensions
				renderer := blackfriday.NewHTMLRenderer(blackfriday.HTMLRendererParameters{
					Flags: blackfriday.CommonHTMLFlags | blackfriday.UseXHTML,
				})
				html := blackfriday.Run([]byte(line), blackfriday.WithExtensions(extensions), blackfriday.WithRenderer(renderer))
				htmlLines = append(htmlLines, string(html))
			} else {
				htmlLines = append(htmlLines, "<p></p>")
			}
		}
	}

	if inTable {
		table := "<table>\n<thead>\n" + tableRows[0] + "\n</thead>\n<tbody>\n"
		table += strings.Join(tableRows[1:], "\n")
		table += "\n</tbody>\n</table>"
		htmlLines = append(htmlLines, table)
	}

	// 添加代码高亮支持
	html := strings.Join(htmlLines, "\n")
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/styles/default.min.css">
    <script src="https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/highlight.min.js"></script>
    <script>hljs.highlightAll();</script>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            line-height: 1.6;
            padding: 20px;
            max-width: 800px;
            margin: 0 auto;
        }
        table {
            border-collapse: separate;
            border-spacing: 0;
            width: 100%%;
            margin: 1em 0;
            border: 1px solid #ddd;
        }
        th, td {
            margin: 0;
            border: 1px solid #ddd;
            padding: 8px;
            white-space: normal;
            word-wrap: break-word;
            overflow-wrap: break-word;
        }
        th {
            background-color: #f8f9fa;
            font-weight: bold;
            border-bottom: 2px solid #ddd;
        }
        tr:nth-child(even) {
            background-color: #f8f9fa;
        }
        td + td, th + th {
            border-left: 1px solid #ddd;
        }
        tr + tr td {
            border-top: 1px solid #ddd;
        }
        [align="center"] {
            text-align: center;
        }
        [align="right"] {
            text-align: right;
        }
        [align="left"] {
            text-align: left;
        }
        blockquote {
            margin: 1em 0;
            padding: 0.5em 1em;
            border-left: 4px solid #ddd;
            background-color: #f9f9f9;
        }
        blockquote > :first-child {
            margin-top: 0;
        }
        blockquote > :last-child {
            margin-bottom: 0;
        }
        blockquote pre {
            background-color: #f0f0f0;
            margin: 0.5em 0;
        }
        blockquote table {
            margin: 0.5em 0;
            background-color: #fff;
        }
        pre {
            background-color: #f5f5f5;
            border: 1px solid #ddd;
            border-radius: 4px;
            padding: 1em;
            margin: 1em 0;
            overflow-x: auto;
            white-space: pre;
            word-wrap: normal;
        }
        pre code {
            background: none;
            border: none;
            padding: 0;
            margin: 0;
            font-family: "SFMono-Regular", Consolas, "Liberation Mono", Menlo, Courier, monospace;
            font-size: 0.9em;
            line-height: 1.4;
            tab-size: 4;
            -moz-tab-size: 4;
        }
        img {
            max-width: 100%%;
            height: auto;
            display: block;
            margin: 1em auto;
            border-radius: 4px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .image-container {
            text-align: center;
            margin: 1em 0;
        }
        .image-container img {
            margin: 0 auto;
        }
        .image-caption {
            color: #666;
            font-size: 0.9em;
            margin-top: 0.5em;
            text-align: center;
        }
    </style>
</head>
<body>
%s
</body>
</html>`, html)
}

func main() {
	// 定义命令行参数
	var inputFile string
	flag.StringVar(&inputFile, "input", "", "输入的Markdown文件路径")
	outputDir := flag.String("output", "", "输出目录")
	flag.Parse()

	// 如果没有使用 -input 参数但提供了位置参数，则第一个位置参数作为输入文件
	if inputFile == "" && flag.NArg() > 0 {
		inputFile = flag.Arg(0)
	}

	if inputFile == "" {
		fmt.Println("用法: markdown2html [文件名] 或 markdown2html -input [文件名] [-output 输出目录]")
		return
	}

	// 获取程序所在目录
	execPath, err := os.Executable()
	if err != nil {
		fmt.Printf("获取程序路径失败: %v\n", err)
		return
	}
	execDir := filepath.Dir(execPath)

	// 读取配置文件
	configPath := filepath.Join(execDir, "config.json")
	configFile, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Printf("读取配置文件失败: %v\n", err)
		return
	}

	// 解析配置文件
	var config Config
	if err := json.Unmarshal(configFile, &config); err != nil {
		fmt.Printf("解析配置文件失败: %v\n", err)
		return
	}

	// 如果未指定输出目录，使用配置文件中的默认值
	outputPath := *outputDir
	if outputPath == "" {
		outputPath = config.Custom.OutputDir
		if outputPath == "" {
			outputPath = "output" // 如果配置文件中也没有指定，使用默认值
		}
	}

	// 如果不是绝对路径，则相对于程序所在目录
	if !filepath.IsAbs(outputPath) {
		// 获取程序所在目录
		execPath, err := os.Executable()
		if err != nil {
			fmt.Printf("获取程序路径失败: %v\n", err)
			return
		}
		execDir := filepath.Dir(execPath)
		outputPath = filepath.Join(execDir, outputPath)
	}

	// 确保输出目录存在
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		fmt.Printf("创建输出目录失败: %v\n", err)
		return
	}

	// 读取Markdown文件
	mdContent, err := ioutil.ReadFile(inputFile)
	if err != nil {
		fmt.Printf("读取文件失败: %v\n", err)
		os.Exit(1)
	}

	// 转换Markdown为HTML
	html := convertMarkdownToHTML(string(mdContent))

	// 生成输出文件名
	baseName := filepath.Base(inputFile)
	outputName := strings.TrimSuffix(baseName, filepath.Ext(baseName)) + ".html"
	outputFile := filepath.Join(outputPath, outputName)

	// 写入HTML文件
	err = ioutil.WriteFile(outputFile, []byte(html), 0644)
	if err != nil {
		fmt.Printf("写入文件失败: %v\n", err)
		os.Exit(1)
	}

	// 上传到MinIO
	fmt.Printf("正在处理文件: %s\n", inputFile)
	customURL, err := uploadToS3(&config, outputFile, []byte(html))
	if err != nil {
		fmt.Printf("上传失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n转换完成！\n")
	fmt.Printf("本地文件: %s\n", outputFile)
	fmt.Printf("访问地址: %s\n", customURL)
}
