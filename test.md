# Markdown 测试文件

## 1. 基本代码块测试

这是一个简单的代码块：

```python
def hello_world():
    print("Hello, World!")
    
    # 带缩进的注释
    for i in range(3):
        print(f"Count: {i}")
```

## 2. 引用测试

> 这是一个简单的引用
> 包含多行内容
> 
> 还可以包含代码块：
> ```go
> func main() {
>     fmt.Println("Hello from quote!")
> }
> ```
> 
> 引用继续...

## 3. 表格测试

| 功能 | 支持情况 | 备注 |
|:---|:---:|---:|
| 代码块 | ✅ | 支持语法高亮 |
| 引用 | ✅ | 支持嵌套元素 |
| 表格 | ✅ | 支持对齐 |

## 4. 复杂嵌套测试

> 这是一个包含表格的引用：
> 
> | 名称 | 描述 |
> |:---|:---|
> | 项目1 | 测试项目 |
> | 项目2 | 另一个测试 |
> 
> 然后是代码块：
> ```javascript
> function test() {
>     // 带缩进的代码
>     console.log("测试缩进");
>     if (true) {
>         console.log("    保留空格");
>     }
> }
> ```
>
> 最后是普通文本
