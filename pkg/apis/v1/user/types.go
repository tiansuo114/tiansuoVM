package user

import "tiansuoVM/pkg/model"

type (
	// 验证码请求和响应
	authCaptchaReq struct{}

	authCaptchaResp struct {
		CaptchaID   string `json:"captcha_id"`
		ImageBase64 string `json:"image_base64"`
	}

	// 登录请求和响应
	loginReq struct {
		UserUID  string `json:"user_uid" binding:"required"`
		Password string `json:"password" binding:"required"`
		//CaptchaID    string `json:"captcha_id" binding:"required"`
		//CaptchaValue string `json:"captcha_value" binding:"required"`
	}

	loginResp struct {
		Token    string `json:"token"`
		UserName string `json:"username"`
	}

	// 用户信息结构体
	UserInfo struct {
		ID        int64            `json:"id"`
		UID       string           `json:"uid"`
		Username  string           `json:"username"`
		Role      model.UserRole   `json:"role"`
		Primary   bool             `json:"primary"`
		Tel       string           `json:"tel"`
		Email     string           `json:"email"`
		Desc      string           `json:"desc"`
		Status    model.UserStatus `json:"status"`
		GidNumber string           `json:"gid_number,omitempty"`
		CreatedAt int64            `json:"created_at"`
		Creator   string           `json:"creator"`
		UpdatedAt int64            `json:"updated_at"`
		Updater   string           `json:"updater"`
	}

	// 当前用户信息响应
	userInfoResp struct {
		User UserInfo `json:"user"`
	}

	// 更新当前用户信息请求
	updateCurrentUserReq struct {
		Tel   string `json:"tel" binding:"omitempty"`
		Email string `json:"email" binding:"omitempty,email"`
		Desc  string `json:"desc" binding:"omitempty"`
	}

	// 更新密码请求 (如果系统支持用户自行修改密码)
	updatePasswordReq struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=6,max=20"`
	}

	// 用户列表请求和响应
	userListReq struct {
		Page  int `form:"page" json:"page"`
		Limit int `form:"limit" json:"limit"`
	}

	userListResp struct {
		Total int64      `json:"total"`
		Users []UserInfo `json:"users"`
	}

	// 更新用户请求
	updateUserReq struct {
		Tel    string           `json:"tel" binding:"omitempty"`
		Email  string           `json:"email" binding:"omitempty,email"`
		Desc   string           `json:"desc" binding:"omitempty"`
		Role   model.UserRole   `json:"role" binding:"omitempty"`
		Status model.UserStatus `json:"status" binding:"omitempty"`
	}

	// 设置用户角色请求
	setRoleReq struct {
		UID  string         `json:"uid" binding:"required"`
		Role model.UserRole `json:"role" binding:"required"`
	}
)
