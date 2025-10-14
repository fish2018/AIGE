# CLAUDE.md - AIGE项目开发配置

## 项目简介
AIGE（AI Game Engine）是一个基于AI驱动的文字冒险游戏引擎，专门用于创建沉浸式的修仙类角色扮演游戏。

## 技术栈
- **前端**: Vue 3 + TypeScript + Element Plus + Pinia + Vite
- **后端**: Go + Gin + GORM + SQLite + WebSocket
- **AI集成**: OpenAI + Anthropic + Google Gemini

## 项目结构
```
AIGE/
├── frontend/           # Vue3前端项目
├── backend/           # Go后端服务
├── mods/              # 游戏MOD系统
├── xiuxian2/          # 修仙游戏模组v2
└── docs/              # 项目文档
```

## 开发环境配置

### 前端开发
```bash
cd frontend
npm install
npm run dev          # 开发服务器 (http://localhost:5173)
npm run build        # 生产构建
npm run type-check   # TypeScript类型检查
npm run lint         # ESLint代码检查
```

### 后端开发  
```bash
cd backend
go mod tidy          # 安装依赖
go run main.go       # 启动服务器 (http://localhost:8182)
```

### 数据库
- 使用SQLite文件数据库 (`backend/chat.db`)
- 自动迁移数据表结构
- 默认创建管理员用户

## 核心功能

### 1. 用户系统
- JWT认证机制
- 角色权限控制（普通用户/管理员）
- 用户注册、登录、资料管理

### 2. 游戏引擎
- **状态管理**: 游戏会话状态持久化
- **AI交互**: 多AI提供商集成，支持流式响应
- **MOD系统**: 可扩展的游戏内容模组
- **对话压缩**: 智能压缩游戏历史保持上下文

### 3. 修仙游戏特色
- **境界体系**: 炼气→筑基→结丹→元婴→化神
- **十次机缘**: 每日游戏次数限制
- **二阶段判定**: AI请求判定→执行→继续叙事
- **动态结局**: 16种结局类型评估
- **宗门系统**: 8大宗门特色玩法
- **资源管理**: 灵石、丹药、法宝、修为

## 主要API接口

### 认证接口
- `POST /api/v1/auth/login` - 用户登录
- `POST /api/v1/auth/register` - 用户注册  
- `GET /api/v1/auth/profile` - 获取用户信息

### 游戏接口
- `GET /api/v1/game/session/:modId` - 获取游戏会话
- `POST /api/v1/game/action` - 处理玩家行动
- `DELETE /api/v1/game/session/:modId` - 删除游戏存档
- `WebSocket /ws/game` - 实时游戏交互

### 管理接口  
- `GET /api/v1/admin/users` - 用户管理
- `GET /api/v1/admin/providers` - AI提供商管理
- `GET /api/v1/admin/models` - AI模型管理

## 配置说明

### AI提供商配置
支持多个AI服务商：
- **OpenAI**: GPT-4, GPT-3.5等模型
- **Anthropic**: Claude系列模型  
- **Google**: Gemini系列模型

### 游戏配置
- **机缘次数**: 每日10次游戏机会
- **自动保存**: 5分钟间隔自动保存
- **压缩机制**: 历史对话智能压缩
- **作弊检测**: AI辅助检测异常行为

## 部署说明

### 开发环境
1. 克隆项目: `git clone <repository>`
2. 启动后端: `cd backend && go run main.go`
3. 启动前端: `cd frontend && npm run dev`
4. 访问: http://localhost:5173

### 生产环境
1. 构建前端: `npm run build`
2. 编译后端: `go build -o aige main.go`
3. 配置环境变量和AI API密钥
4. 启动服务: `./aige`

## 开发规范

### 提交格式
```
feat: 新功能
fix: 修复bug  
docs: 文档更新
style: 代码格式
refactor: 重构
test: 测试相关
chore: 构建/工具变动
```

### 代码规范
- **Go**: 遵循gofmt和golint规范
- **TypeScript**: 使用ESLint + Prettier
- **Vue**: 使用Composition API风格
- **命名**: 驼峰命名，语义化变量名

## 故障排查

### 常见问题
1. **端口冲突**: 前端5173，后端8182
2. **跨域问题**: 已配置CORS，检查端口号
3. **数据库锁定**: 重启后端服务
4. **AI调用失败**: 检查API密钥和网络连接

### 日志查看
- 后端日志: 控制台输出
- 前端日志: 浏览器开发者工具
- 游戏状态: 数据库chat.db中的game_saves表

## 扩展开发

### 添加新MOD
1. 在`mods/`目录创建新文件夹
2. 添加`config.json`配置文件  
3. 在`prompts/`目录添加提示词文件
4. 重启后端服务加载新MOD

### 接入新AI提供商
1. 在`services/ai_client.go`添加新的调用方法
2. 在`controllers/provider.go`添加配置支持
3. 更新前端管理界面
4. 测试API调用和流式响应

### 自定义游戏功能
1. 修改游戏状态结构（`game_engine/state_manager.go`）
2. 扩展游戏控制器（`game_engine/game_controller.go`）  
3. 更新前端游戏界面
4. 添加相关的API接口

## 性能优化建议

### 后端优化
- 使用数据库连接池
- 实现Redis缓存层
- 优化AI调用频率
- 异步处理长时间任务

### 前端优化  
- 组件懒加载
- 虚拟滚动列表
- 状态缓存管理
- 资源压缩和CDN

## 安全注意事项

### API安全
- 所有接口使用JWT认证
- 敏感操作需要管理员权限
- 输入参数校验和过滤
- SQL注入防护（使用GORM）

### 业务安全
- 游戏状态完整性校验
- 作弊行为AI检测
- 用户行为异常监控
- API访问频率限制

## 测试策略

### 单元测试
```bash
# 后端测试
go test ./...

# 前端测试  
npm test
```

### 集成测试
- API接口测试
- 数据库操作测试
- AI服务集成测试
- WebSocket连接测试

### E2E测试
- 完整游戏流程测试
- 用户注册登录流程
- 管理员功能测试
- 跨浏览器兼容性测试

## 监控和日志

### 应用监控
- API响应时间
- 错误率统计
- 数据库性能
- 内存使用情况

### 业务监控
- 用户活跃度
- 游戏会话统计  
- AI调用成本
- 功能使用热图

## 版本历史

### v1.0.0 (当前版本)
- 基础游戏引擎实现
- 修仙游戏MOD完成
- 多AI提供商支持
- 流式对话交互
- 用户权限管理系统

## 联系方式
- 项目维护: AIGE Team
- 问题反馈: GitHub Issues
- 功能建议: 项目讨论区

---

**注意**: 该项目仍在积极开发中，API和功能可能会发生变化。请关注项目更新和版本发布说明。