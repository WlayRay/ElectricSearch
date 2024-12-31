package infrastructure

import "strings"

const (
	BIAN_CHENG = 1 << iota
	CHENG_XU_YUAN
	GUI_CHU
	JI_LU
	KE_JI
	MEI_SHI
	YIN_YUE
	YING_SHI
	YU_LE
	YU_XI
	ZONG_YI
	ZHI_SHI
	ZI_XUN
	FAN_JU
	YOU_XI
)

func GetCategoriesBits(keywords []string) uint64 {
	var bits uint64
	for _, keyword := range keywords {
		switch strings.ToLower(keyword) {
		case "编程":
			bits |= BIAN_CHENG
		case "鬼畜":
			bits |= GUI_CHU
		case "纪录":
			bits |= JI_LU
		case "科技":
			bits |= KE_JI
		case "美食":
			bits |= MEI_SHI
		case "音乐":
			bits |= YIN_YUE
		case "影视":
			bits |= YING_SHI
		case "娱乐":
			bits |= YU_LE
		case "游戏":
			bits |= YU_XI
		case "综艺":
			bits |= ZONG_YI
		case "知识":
			bits |= ZHI_SHI
		case "资讯":
			bits |= ZI_XUN
		case "游记":
			bits |= YOU_XI
		case "程序员":
			bits |= CHENG_XU_YUAN
		default:
			// 不匹配任何关键词，不做处理
		}
	}

	return bits
}
