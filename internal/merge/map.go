package merge

import "strings"

// Field describes how a material entry (label) maps onto a vault field.
// FieldID is the template without the person prefix, e.g. "identity.email".
// The final stored id is "<personPrefix>_<FieldID>" (e.g. "p1_identity.email").
type Field struct {
	FieldID     string   // "category.field"
	Label       string   // canonical Chinese label
	Aliases     []string // alternative labels / synonyms
	Sensitivity string   // default sensitivity for the stored value
	Cardinality int      // 0 = single value, 1 = list member
}

// fieldMap drives the label -> field resolution for merge.
var fieldMap = []Field{
	// ---- identity ----
	{FieldID: "identity.name", Label: "姓名", Aliases: []string{"名字"}, Sensitivity: "standard"},
	{FieldID: "identity.gender", Label: "性别", Sensitivity: "standard"},
	{FieldID: "identity.date_of_birth", Label: "出生日期", Aliases: []string{"出生年月", "生日"}, Sensitivity: "sensitive"},
	{FieldID: "identity.ethnicity", Label: "民族", Sensitivity: "standard"},
	{FieldID: "identity.native_place", Label: "籍贯", Sensitivity: "standard"},
	{FieldID: "identity.political_status", Label: "政治面貌", Sensitivity: "standard"},
	{FieldID: "identity.marital_status", Label: "婚姻状况", Sensitivity: "standard"},
	{FieldID: "identity.id_number", Label: "证件号码", Aliases: []string{"身份证号", "证件号", "身份证号码"}, Sensitivity: "critical"},
	{FieldID: "identity.phone", Label: "手机号", Aliases: []string{"手机号码", "联系电话"}, Sensitivity: "sensitive"},
	{FieldID: "identity.alt_phone", Label: "其他联系方式", Aliases: []string{"备用手机号", "备用电话"}, Sensitivity: "sensitive"},
	{FieldID: "identity.email", Label: "邮箱", Aliases: []string{"电子邮箱", "email"}, Sensitivity: "standard"},
	{FieldID: "identity.email_backup", Label: "备用邮箱", Sensitivity: "standard"},
	{FieldID: "identity.height", Label: "身高", Sensitivity: "standard"},
	{FieldID: "identity.weight", Label: "体重", Sensitivity: "standard"},
	{FieldID: "identity.nationality", Label: "国籍", Sensitivity: "standard"},
	{FieldID: "identity.health_status", Label: "健康状况", Sensitivity: "standard"},
	{FieldID: "identity.household_location", Label: "户口所在地", Aliases: []string{"户籍所在地", "户口地址"}, Sensitivity: "sensitive"},
	{FieldID: "identity.birth_place", Label: "生源地", Sensitivity: "standard"},
	{FieldID: "identity.foreign_permission", Label: "是否取得外国国籍或国（境）外永久居留资格", Aliases: []string{"外国国籍或永久居留"}, Sensitivity: "standard"},
	{FieldID: "identity.school_household", Label: "现户口是否为学校户口", Sensitivity: "standard"},
	{FieldID: "identity.has_children", Label: "有无子女", Sensitivity: "standard"},

	// ---- career_objective ----
	{FieldID: "career_objective.target_city", Label: "期待工作地点", Aliases: []string{"期望城市", "工作地点"}, Sensitivity: "standard"},
	{FieldID: "career_objective.target_position", Label: "期望职位", Aliases: []string{"求职意向职位"}, Sensitivity: "standard"},
	{FieldID: "career_objective.graduate_time", Label: "毕业时间", Sensitivity: "standard"},
	{FieldID: "career_objective.graduate_type", Label: "人员类型", Aliases: []string{"生源类型"}, Sensitivity: "standard"},
	{FieldID: "career_objective.adjust_ok", Label: "是否同意调剂到其他岗位", Aliases: []string{"是否同意调剂"}, Sensitivity: "standard"},
	{FieldID: "career_objective.contact_address", Label: "通讯地址", Aliases: []string{"联系地址"}, Sensitivity: "sensitive"},

	// ---- education (list) ----
	{FieldID: "education.postgrad_school", Label: "硕士院校", Sensitivity: "standard", Cardinality: 1},
	{FieldID: "education.postgrad_college", Label: "硕士学院", Sensitivity: "standard", Cardinality: 1},
	{FieldID: "education.postgrad_major", Label: "硕士专业", Sensitivity: "standard", Cardinality: 1},
	{FieldID: "education.postgrad_category", Label: "硕士专业类别", Sensitivity: "standard", Cardinality: 1},
	{FieldID: "education.postgrad_degree", Label: "硕士学历", Sensitivity: "standard", Cardinality: 1},
	{FieldID: "education.postgrad_degree_name", Label: "硕士学位", Sensitivity: "standard", Cardinality: 1},
	{FieldID: "education.postgrad_period", Label: "硕士起止时间", Aliases: []string{"硕士开始时间", "硕士结束时间"}, Sensitivity: "standard", Cardinality: 1},
	{FieldID: "education.postgrad_form", Label: "硕士学习形式", Sensitivity: "standard", Cardinality: 1},
	{FieldID: "education.undergrad_school", Label: "本科院校", Sensitivity: "standard", Cardinality: 1},
	{FieldID: "education.undergrad_college", Label: "本科学院", Sensitivity: "standard", Cardinality: 1},
	{FieldID: "education.undergrad_major", Label: "本科专业", Sensitivity: "standard", Cardinality: 1},
	{FieldID: "education.undergrad_category", Label: "本科专业类别", Sensitivity: "standard", Cardinality: 1},
	{FieldID: "education.undergrad_degree", Label: "本科学历", Sensitivity: "standard", Cardinality: 1},
	{FieldID: "education.undergrad_degree_name", Label: "本科学位", Sensitivity: "standard", Cardinality: 1},
	{FieldID: "education.undergrad_period", Label: "本科起止时间", Aliases: []string{"本科开始时间", "本科结束时间"}, Sensitivity: "standard", Cardinality: 1},
	{FieldID: "education.undergrad_form", Label: "本科学习形式", Sensitivity: "standard", Cardinality: 1},

	// ---- career (list: internship + project) ----
	{FieldID: "career.internship_org", Label: "实习单位", Aliases: []string{"实习单位名称"}, Sensitivity: "standard", Cardinality: 1},
	{FieldID: "career.internship_period", Label: "实习起止", Sensitivity: "standard", Cardinality: 1},
	{FieldID: "career.internship_position", Label: "实习职位", Sensitivity: "standard", Cardinality: 1},
	{FieldID: "career.project_name", Label: "项目名称", Sensitivity: "standard", Cardinality: 1},
	{FieldID: "career.project_period", Label: "项目时间", Sensitivity: "standard", Cardinality: 1},
	{FieldID: "career.project_desc", Label: "项目描述", Sensitivity: "standard", Cardinality: 1},

	// ---- awards (list) ----
	{FieldID: "awards.student_leader", Label: "学生干部任职", Sensitivity: "standard", Cardinality: 1},
	{FieldID: "awards.math_contest", Label: "数学竞赛获奖", Sensitivity: "standard", Cardinality: 1},
	{FieldID: "awards.sandbox_contest", Label: "沙盘竞赛获奖", Sensitivity: "standard", Cardinality: 1},
	{FieldID: "awards.best_thesis", Label: "优秀毕业论文", Sensitivity: "standard", Cardinality: 1},
	{FieldID: "awards.scholarship", Label: "奖学金情况", Sensitivity: "standard", Cardinality: 1},

	// ---- family (list) ----
	{FieldID: "family.mother_name", Label: "母亲姓名", Sensitivity: "standard", Cardinality: 1},
	{FieldID: "family.mother_political", Label: "母亲政治面貌", Sensitivity: "standard", Cardinality: 1},
	{FieldID: "family.mother_org", Label: "母亲工作单位", Sensitivity: "standard", Cardinality: 1},
	{FieldID: "family.sister_name", Label: "妹妹姓名", Aliases: []string{"兄弟姐妹姓名"}, Sensitivity: "standard", Cardinality: 1},
	{FieldID: "family.sister_political", Label: "妹妹政治面貌", Sensitivity: "standard", Cardinality: 1},
	{FieldID: "family.sister_org", Label: "妹妹工作单位", Sensitivity: "standard", Cardinality: 1},
	{FieldID: "family.kinship_declaration", Label: "亲属声明", Sensitivity: "standard", Cardinality: 1},

	// ---- skills ----
	{FieldID: "skills.language_chinese", Label: "中文", Sensitivity: "standard"},
	{FieldID: "skills.language_english", Label: "英语水平", Aliases: []string{"英语", "英语等级"}, Sensitivity: "standard"},
	{FieldID: "skills.hobby_ai", Label: "AI技能/兴趣", Aliases: []string{"技能/兴趣爱好"}, Sensitivity: "standard"},
	{FieldID: "skills.hobby_office", Label: "办公技能/兴趣", Sensitivity: "standard"},
}

// ResolveField finds the merge field definition by label or alias.
func ResolveField(label string) *Field {
	for i := range fieldMap {
		f := &fieldMap[i]
		if strings.EqualFold(f.Label, label) {
			return f
		}
		for _, al := range f.Aliases {
			if strings.EqualFold(al, label) {
				return f
			}
		}
	}
	return nil
}

// SensitivityWeight maps a sensitivity tier to a numeric weight used for the
// three-tier classification (higher = more sensitive = more confirmation).
func SensitivityWeight(s string) int {
	switch s {
	case "public":
		return 0
	case "standard":
		return 1
	case "sensitive":
		return 2
	case "critical":
		return 3
	}
	return 1
}