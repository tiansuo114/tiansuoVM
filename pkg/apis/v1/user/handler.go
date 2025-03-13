package user

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"tiansuoVM/pkg/helper"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"tiansuoVM/pkg/apis/v1/logs"
	"tiansuoVM/pkg/captcha"
	"tiansuoVM/pkg/client/ldap"
	"tiansuoVM/pkg/dao"
	"tiansuoVM/pkg/dbresolver"
	"tiansuoVM/pkg/model"
	"tiansuoVM/pkg/server/encoding"
	"tiansuoVM/pkg/server/errutil"
	"tiansuoVM/pkg/server/request"
	"tiansuoVM/pkg/token"
	"tiansuoVM/pkg/types"
	"tiansuoVM/pkg/utils"
	"tiansuoVM/pkg/utils/limiter"
)

var (
	// 验证码限制器
	captchaLimiter = limiter.NewKeyLimiter(1, 1)
	// 登录限制器
	loginLimiter = limiter.NewLoginLimiter()
)

const (
	// 登录失败次数阈值
	loginFailThreshold = 5
	// IP验证码限制键前缀
	captchaIPPrefix = "ip:"
	// 用户名验证码限制键前缀
	captchaUsernamePrefix = "user:"
	// 登录限制键前缀
	loginIPPrefix = "login_ip:"
	// 用户名登录限制键前缀
	loginUsernamePrefix = "login_user:"
)

type handlerOption struct {
	tokenManager   token.Manager
	dbResolver     *dbresolver.DBResolver
	ldapClient     *ldap.LDAPClient
	captchaLimiter *limiter.KeyLimiter
	loginLimiter   *limiter.LoginLimiter
}

type handler struct {
	handlerOption
}

func newHandler(option handlerOption) *handler {
	return &handler{
		handlerOption: option,
	}
}

// authCaptcha 生成验证码
func (h *handler) authCaptcha(c *gin.Context) {
	// 限制验证码请求频率
	if !h.captchaLimiter.AllowKey(utils.MD5Hex(c.Request.UserAgent())) {
		encoding.HandleError(c, errutil.NewError(http.StatusTooManyRequests, "请求验证码过于频繁"))
		return
	}

	// 创建验证码
	captchaID, imageBase64, err := captcha.CreateCaptcha()
	if err != nil {
		zap.L().Error("生成验证码失败", zap.Error(err))
		encoding.HandleError(c, errutil.NewError(http.StatusInternalServerError, "生成验证码失败"))
		return
	}

	// 返回验证码信息
	encoding.HandleSuccess(c, authCaptchaResp{
		CaptchaID:   captchaID,
		ImageBase64: imageBase64,
	})
}

// login 用户登录
func (h *handler) login(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c, types.DefaultTimeout)
	defer cancel()
	tokenUser := token.GetUIDFromCtx(c)

	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		zap.L().Error("解析登录请求失败", zap.Error(err))
		encoding.HandleError(c, errutil.ErrIllegalParameter)
		return
	}

	// Check if there is already an active token in the request context
	if t, err := h.tokenManager.GetTokenFromCtx(c); err == nil {
		if clm, err := h.tokenManager.Verify(t); err == nil {
			encoding.HandleSuccess(c, loginResp{
				UserName: clm.Name,
				Token:    t,
			})
			return
		}
	}

	// 清理输入
	req.UserUID = strings.TrimSpace(req.UserUID)
	req.Password = strings.TrimSpace(req.Password)
	//req.CaptchaValue = strings.TrimSpace(req.CaptchaValue)

	// 验证请求
	if err := request.ValidateStruct(ctx, req); err != nil {
		encoding.HandleError(c, err)
		return
	}

	//// 验证码检查
	//if !captcha.VerifyCaptcha(req.CaptchaID, strings.ToLower(req.CaptchaValue)) {
	//	encoding.HandleError(c, errutil.NewError(http.StatusBadRequest, "验证码错误"))
	//	return
	//}

	// 查找LDAP用户
	ldapUser, err := h.ldapClient.FindUserByUID(req.UserUID)
	if err != nil {
		zap.L().Error("LDAP查找用户失败", zap.Error(err), zap.String("username", req.UserUID))
		encoding.HandleError(c, errutil.NewError(http.StatusBadRequest, "用户不存在"))
		return
	}

	// 验证密码
	if err = h.ldapClient.Bind(ldapUser.DN, req.Password); err != nil {
		zap.L().Error("LDAP密码验证失败", zap.Error(err), zap.String("username", req.UserUID))
		encoding.HandleError(c, errutil.NewError(http.StatusBadRequest, "用户名或密码错误"))
		return
	}

	// 检查用户是否存在于数据库
	found, user, err := dao.GetUserByUID(ctx, h.dbResolver, ldapUser.UID)
	if err != nil {
		zap.L().Error("查询用户失败", zap.Error(err), zap.String("uid", ldapUser.UID))
		encoding.HandleError(c, errutil.ErrInternalServer)
		return
	}

	var tokenInfo token.Info

	if found && user != nil {
		// 用户已存在
		tokenInfo = token.Info{
			UID:      user.UID,
			Username: user.Username,
			Name:     user.Username,
			Role:     user.Role,
			Primary:  true,
		}

		// 记录登录日志
		logs.UserOperatorLogChannel <- &model.UserOperatorLog{
			UID:       user.UID,
			Operator:  model.UserOperatorLogin,
			CreatedAt: time.Now().UnixMilli(),
			Creator:   user.UID,
		}
	} else {
		err = h.dbResolver.GetDB().Transaction(func(tx *gorm.DB) error {
			// 设置用户角色，根据LDAP信息决定
			userRole := model.UserRoleNormal
			if helper.LDAPGroupCache[ldapUser.GIDNumber] == string(model.UserRoleAdmin) {
				userRole = model.UserRoleAdmin
			}

			// 创建新用户
			newUser := model.User{
				UID:       ldapUser.UID,
				Username:  "默认名称",
				Tel:       ldapUser.TelephoneNumber,
				Email:     ldapUser.Mail,
				Desc:      "LDAP用户",
				Role:      userRole,
				Status:    model.UserStatusEnabled,
				GidNumber: ldapUser.GIDNumber,
				Primary:   true,
			}

			if err := dao.CreateUser(ctx, h.dbResolver, &newUser); err != nil {
				return err
			}

			user = &newUser
			tokenInfo = token.Info{
				UID:      newUser.UID,
				Username: newUser.Username,
				Name:     newUser.Username,
				Role:     newUser.Role,
				Primary:  true,
			}

			// 记录首次登录日志
			logs.UserOperatorLogChannel <- &model.UserOperatorLog{
				UID:       newUser.UID,
				Operator:  model.UserOperatorFirstLogin,
				CreatedAt: time.Now().UnixMilli(),
				Creator:   newUser.UID,
			}

			return nil
		})

		if err != nil {
			zap.L().Error("创建新用户失败", zap.Error(err))
			encoding.HandleError(c, errutil.ErrInternalServer)
			return
		}
	}

	// 生成token
	tokenStr, err := h.tokenManager.IssueTo(tokenInfo, token.DefaultCacheDuration)
	if err != nil {
		zap.L().Error("生成token失败", zap.Error(err))
		encoding.HandleError(c, errutil.NewError(http.StatusInternalServerError, "生成登录凭证失败"))
		return
	}

	// 返回登录结果
	encoding.HandleSuccess(c, loginResp{
		Token:    tokenStr,
		UserName: user.Username,
	})
	defer func() {
		if err != nil {
			logs.UserOperatorLogChannel <- &model.UserOperatorLog{
				UID:       req.UserUID,
				Operator:  model.UserOperatorError, // Store the JSON string as operator details
				Operation: fmt.Sprintf("user login error : %s", err),
				CreatedAt: time.Now().UnixMilli(),
				Creator:   tokenUser, // Creator of the operation
			}
		}
	}()
}

// getCurrentUser 获取当前用户信息
func (h *handler) getCurrentUser(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c, types.DefaultTimeout)
	defer cancel()

	// 从Token中获取用户ID
	uid := token.GetUIDFromCtx(c)
	if uid == "" {
		encoding.HandleError(c, errutil.ErrUnauthorized)
		return
	}

	// 查询用户信息
	found, user, err := dao.GetUserByUID(ctx, h.dbResolver, uid)
	if err != nil {
		zap.L().Error("查询用户信息失败", zap.Error(err), zap.String("uid", uid))
		encoding.HandleError(c, errutil.ErrInternalServer)
		return
	}

	if !found || user == nil {
		encoding.HandleError(c, errutil.NewError(http.StatusNotFound, "用户不存在"))
		return
	}

	userInfoResp := UserInfo{
		ID:        user.ID,
		UID:       user.UID,
		Username:  user.Username,
		Role:      user.Role,
		Primary:   false,
		Tel:       user.Tel,
		Email:     user.Email,
		Desc:      user.Desc,
		Status:    user.Status,
		CreatedAt: user.CreatedAt,
		Creator:   user.Creator,
		UpdatedAt: user.UpdatedAt,
		Updater:   user.Updater,
	}

	// 返回用户信息
	encoding.HandleSuccess(c, userInfoResp)
}

// updateCurrentUser 更新当前用户信息
func (h *handler) updateCurrentUser(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c, types.DefaultTimeout)
	defer cancel()

	// 从Token中获取用户ID
	uid := token.GetUIDFromCtx(c)
	if uid == "" {
		encoding.HandleError(c, errutil.ErrUnauthorized)
		return
	}

	var req updateCurrentUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		zap.L().Error("解析更新请求失败", zap.Error(err))
		encoding.HandleError(c, errutil.ErrIllegalParameter)
		return
	}

	// 验证请求
	if err := request.ValidateStruct(ctx, req); err != nil {
		encoding.HandleError(c, err)
		return
	}

	// 检查用户是否存在
	found, _, err := dao.GetUserByUID(ctx, h.dbResolver, uid)
	if err != nil {
		zap.L().Error("查询用户失败", zap.Error(err), zap.String("uid", uid))
		encoding.HandleError(c, errutil.ErrInternalServer)
		return
	}

	if !found {
		encoding.HandleError(c, errutil.NewError(http.StatusNotFound, "用户不存在"))
		return
	}

	// 构建更新内容
	updates := make(map[string]interface{})
	if req.Tel != "" {
		updates["tel"] = req.Tel
	}
	if req.Email != "" {
		updates["email"] = req.Email
	}
	if req.Desc != "" {
		updates["desc"] = req.Desc
	}

	// 如果没有任何需要更新的字段
	if len(updates) == 0 {
		encoding.HandleSuccess(c)
		return
	}

	// 执行更新
	if err := dao.UpdateUser(ctx, h.dbResolver, uid, updates); err != nil {
		zap.L().Error("更新用户信息失败", zap.Error(err), zap.String("uid", uid))
		encoding.HandleError(c, errutil.ErrInternalServer)
		return
	}

	// 记录操作日志
	logs.UserOperatorLogChannel <- &model.UserOperatorLog{
		UID:       uid,
		Operator:  model.UserOperatorUpdate,
		Operation: fmt.Sprintf("用户更新个人信息"),
		CreatedAt: time.Now().UnixMilli(),
		Creator:   uid,
	}

	encoding.HandleSuccess(c)
}

// logout 用户登出
func (h *handler) logout(c *gin.Context) {
	// 简单地返回成功，实际的token失效处理可以在中间件中完成
	encoding.HandleSuccess(c)
}
