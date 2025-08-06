# Generate-PB: FIX to Protobuf Code Generator

这个工具扩展了原有的 `generate-pb` 功能，添加了FIX消息结构到Protobuf的转换功能。

## 功能概述

### 原有功能
- 解析FIX数据字典（XML文件）
- 生成Protobuf消息定义（.proto文件）
- 生成枚举帮助函数

### 新增功能
- **FIX到Protobuf转换**：为每个FIX消息类型生成从QuickFIX/Go消息到Protobuf消息的转换函数
- **枚举转换**：自动处理FIX枚举值与Protobuf枚举之间的转换
- **字段类型映射**：正确处理不同FIX字段类型到Protobuf类型的映射

## 使用方法

```bash
./generate-pb -pb_go_pkg <go_package> -pb_root <proto_dir> -go_root <go_dir> -fix_pkg <fix_package> [flags] <fix_spec.xml>
```

### 参数说明

- `-pb_go_pkg`: 生成的protobuf Go包路径
- `-pb_root`: protobuf文件输出目录
- `-go_root`: Go转换代码输出目录  
- `-fix_pkg`: QuickFIX/Go包的根导入路径
- `-verbose`: 启用详细输出
- `-dry-run`: 执行预演，不实际写入文件

### 示例

```bash
./generate-pb \
  -pb_go_pkg github.com/mycompany/proto \
  -pb_root ./proto \
  -go_root ./internal/proto \
  -fix_pkg github.com/quickfixgo/quickfix \
  -verbose \
  spec/FIX42.xml
```

## 生成的文件

### 1. fix.g.proto
包含所有FIX消息的Protobuf定义：
- 消息结构定义
- 枚举类型定义
- 重复组（Repeating Groups）定义

### 2. enum_helpers.go
包含枚举转换的辅助函数：
- `<EnumType>ToString`: Protobuf枚举到FIX字符串映射
- `StringTo<EnumType>`: FIX字符串到Protobuf枚举映射

### 3. fix_to_proto.go
包含消息转换函数：
- `<MessageType>FromFIX()`: 将QuickFIX/Go消息转换为Protobuf消息
- 枚举转换函数：`Convert<EnumType>FromFIX()` 和 `Convert<EnumType>ToFIX()`

## 支持的字段类型转换

| FIX字段类型 | Protobuf类型 | 转换说明 |
|------------|-------------|----------|
| INT, SEQNUM | int32 | 32位整数 |
| LENGTH, TAGNUM | uint32 | 32位无符号整数 |
| FLOAT | float64 | 64位浮点数 |
| BOOLEAN | bool | 布尔值（Y/N -> true/false） |
| PRICE, QTY, AMT | string | 保持字符串以维持精度 |
| STRING, CHAR | string | 字符串类型 |
| ENUM | enum | 自定义枚举类型 |

## 代码使用示例

```go
package main

import (
    "fmt"
    "github.com/quickfixgo/quickfix"
    "github.com/test/proto" // 生成的包
)

func main() {
    // 假设你有一个QuickFIX/Go消息
    var fixMessage quickfix.Messagable
    
    // 转换为Protobuf消息
    protoMsg, err := proto.LogoutFromFIX(fixMessage)
    if err != nil {
        fmt.Printf("转换失败: %v\n", err)
        return
    }
    
    // 现在可以使用protobuf消息
    fmt.Printf("转换后的消息: %+v\n", protoMsg)
}
```

## 枚举转换示例

```go
// FIX枚举值转Protobuf枚举
side := proto.ConvertSideFromFIX("1") // "1" -> Side_BUY

// Protobuf枚举转FIX值
fixValue := proto.ConvertSideToFIX(proto.Side_BUY) // Side_BUY -> "1"
```

## 注意事项

1. **组件转换**：当前版本对复杂组件的转换支持有限，需要根据具体消息结构进行额外实现
2. **重复组**：基本的重复组转换已实现，但复杂嵌套结构可能需要手动调整
3. **精度保持**：价格和数量字段使用字符串类型以保持精度
4. **错误处理**：转换函数包含基本的错误处理，建议在生产环境中增加更全面的验证

## 架构对比

### generate-fix vs generate-pb

| 功能 | generate-fix | generate-pb (新版) |
|------|-------------|-------------------|
| 输出格式 | Go结构体 | Protobuf + 转换函数 |
| 消息表示 | QuickFIX/Go原生 | 语言无关的Protobuf |
| 跨语言支持 | 仅Go | 支持多种语言 |
| 序列化 | FIX协议 | Protobuf二进制 |
| 类型安全 | Go类型系统 | Protobuf类型系统 |
| 转换开销 | 无 | 轻微的转换开销 |

这个扩展版本使你能够在保持QuickFIX/Go强大FIX协议处理能力的同时，享受Protobuf的跨语言、高效序列化等优势。
