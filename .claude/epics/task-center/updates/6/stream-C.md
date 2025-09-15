---
issue: 6
stream: 业务系统管理功能
agent: general-purpose
started: 2025-09-15T08:17:44Z
completed: 2025-09-15T18:30:00Z
status: completed
depends_on: [stream-A, stream-B]
---

# Stream C: 业务系统管理功能

## Scope
实现业务系统注册、管理、状态控制和配额设置功能

## Files
- ✅ internal/logic/business/system_logic.go (新增)
- ✅ internal/handler/business/system_handler.go (新增)
- ✅ internal/logic/business/apikey_management_logic.go (新增)
- ✅ internal/handler/business/apikey_management_handler.go (新增)
- ✅ internal/handler/business_management_routes.go (新增)
- ✅ internal/model/businesssystemsmodel.go (扩展)

## Completed Features

### 1. 业务系统管理逻辑 ✅
- **文件**: `internal/logic/business/system_logic.go`
- **功能**:
  - 业务系统注册：自动生成API Key和Secret
  - 业务系统信息查询和列表
  - 业务系统状态管理（active/disabled/suspended）
  - 业务系统信息更新
  - 配额设置和管理
  - API Key重新生成功能
  - 集成Stream A和B的认证机制

### 2. 业务系统管理处理器 ✅
- **文件**: `internal/handler/business/system_handler.go`
- **功能**:
  - 注册业务系统处理器
  - 获取业务系统信息处理器
  - 业务系统列表处理器（分页）
  - 更新业务系统状态处理器
  - 更新业务系统信息处理器
  - 设置业务系统配额处理器
  - 重新生成API Key处理器

### 3. API Key管理逻辑 ✅
- **文件**: `internal/logic/business/apikey_management_logic.go`
- **功能**:
  - 为业务系统创建API Key
  - API Key信息查询和验证
  - API Key列表管理
  - API Key轮换机制
  - API Key撤销功能
  - API Key权限验证
  - API Key使用统计（框架）
  - 重用Stream B的核心认证逻辑

### 4. API Key管理处理器 ✅
- **文件**: `internal/handler/business/apikey_management_handler.go`
- **功能**:
  - 创建API Key处理器
  - 获取API Key信息处理器
  - API Key列表处理器
  - 轮换API Key处理器
  - 撤销API Key处理器
  - 验证API Key权限处理器
  - 获取API Key使用统计处理器

### 5. 数据模型扩展 ✅
- **文件**: `internal/model/businesssystemsmodel.go`
- **改进**:
  - 添加分页查询方法：QueryPagedSystems
  - 添加计数方法：CountSystems
  - 支持自定义查询和统计

### 6. 路由配置集成 ✅
- **文件**: `internal/handler/business_management_routes.go`
- **新增路由**:
  - 管理员路由（需JWT认证）：
    - `POST /api/v1/admin/systems/register` - 注册业务系统
    - `GET /api/v1/admin/systems` - 获取业务系统列表
    - `GET /api/v1/admin/systems/info` - 获取业务系统信息
    - `PATCH /api/v1/admin/systems/status` - 更新业务系统状态
    - `PUT /api/v1/admin/systems/info` - 更新业务系统信息
    - `PUT /api/v1/admin/systems/quota` - 设置业务系统配额
    - `POST /api/v1/admin/systems/regenerate-apikey` - 重新生成API Key
  - 业务系统路由（需JWT认证）：
    - `POST /api/v1/business/apikeys/create` - 创建API Key
    - `GET /api/v1/business/apikeys/info` - 获取API Key信息
    - `GET /api/v1/business/apikeys/list` - 列出API Key
    - `POST /api/v1/business/apikeys/rotate` - 轮换API Key
    - `DELETE /api/v1/business/apikeys/revoke` - 撤销API Key
    - `GET /api/v1/business/apikeys/stats` - 获取API Key使用统计
  - 公开验证路由（无需认证）：
    - `GET /api/v1/auth/validate-permission` - 验证API Key权限

## Technical Implementation

### 业务系统注册流程
1. 验证业务系统代码唯一性
2. 生成API Key（tc_[32位随机字符]）和Secret（64位随机字符）
3. 使用bcrypt加密存储Secret
4. 设置默认配额和状态
5. 存储联系人信息

### API Key管理
- 重用Stream B的API Key生成和验证逻辑
- 支持业务系统自主管理API Key
- 提供轮换和撤销机制
- 集成权限验证系统

### 状态管理
- 状态映射：0-disabled, 1-active, 2-suspended
- 状态变更记录和原因
- 状态检查集成到所有认证流程

### 配额管理
- 速率限制（requests/min）
- 支持动态调整
- 为后续监控和限流做准备

## Integration with Stream A/B
- ✅ 重用Stream A的JWT认证机制
- ✅ 集成Stream B的API Key生成和验证逻辑
- ✅ 使用Stream B的权限管理系统
- ✅ 保持与现有认证流程的兼容性

## Next Steps for Stream D
Stream C已完成业务系统管理功能，Stream D可以开始权限控制工作：
- 基于业务系统的数据隔离
- 细粒度权限控制
- 资源访问限制
- 安全审计功能

## Commit
```
Issue #6: 实现业务系统管理和API Key管理功能
- 完整的业务系统CRUD操作
- API Key生成、轮换、撤销机制
- 状态和配额管理
- 集成Stream A/B认证机制
- 管理员和业务系统双重API接口
```