# 凡语言

凡语言是一门以中文关键字为核心的通用脚本语言，使用 Go 实现。它借鉴 Go 的设计哲学，强调简单、直接、组合优于继承，同时保留脚本语言的轻量和易用。

## 特性

- 中文关键字与符号运算符并存
- 变量、常量与可选类型标注
- 算术、比较、逻辑运算和三元表达式
- `如果`、`当`、`重复`、`遍历` 等控制结构
- 一等函数、闭包、递归、多返回值
- 默认参数与可变参数
- 模型、嵌入组合、方法扩展与鸭子类型
- Go 风格错误值、`检查` 与 `尝试…捕获`
- 真实模型标签，支持 Go 与凡语言处理器
- 基于文件的模块导入和导出
- 单二进制 CLI，支持脚本运行与交互式 REPL

## 快速示例

```凡
变量 名字 = "世界"
打印("你好，", 名字)

定义 函数 求和(甲, 乙)
    返回 甲 加 乙
结束

打印(求和(2, 3))

函数 多值求和(开头, 其余 ...整数)
    变量 合计 = 开头
    遍历 其余 中的 数
        合计 += 数
    结束
    返回 合计
结束

打印(多值求和(1, 2, 3, 4))
```

## 安装

### Linux / macOS

```sh
curl -fsSL https://raw.githubusercontent.com/china-lang-fan/fan/main/scripts/install.sh | bash
```

### Windows

```powershell
irm https://raw.githubusercontent.com/china-lang-fan/fan/main/scripts/install.ps1 | iex
```

也可以从 [GitHub Releases](https://github.com/china-lang-fan/fan/releases) 下载对应平台的压缩包。

安装后验证：

```sh
fan version
```

## 使用

运行脚本：

```sh
fan run 脚本.凡
```

进入 REPL：

```sh
fan repl
```

查看帮助：

```sh
fan help
```

更多示例见 [`examples/`](./examples)，完整文档见 <https://china-lang-fan.github.io/>。

## 从源码构建

需要 Go 1.26 或更高版本。

```sh
go build ./cmd/fan
go test ./...
```

## 编辑器支持

- [VS Code 插件](https://github.com/china-lang-fan/fan-code-plugin)：语法高亮、代码片段、注释切换、运行脚本和 REPL
- JetBrains 插件：见 [`fan-jetbrains-plugin`](../fan-jetbrains-plugin)

## 许可证

MIT
