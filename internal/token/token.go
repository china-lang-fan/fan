package token

type Type string

const (
	ILLEGAL Type = "ILLEGAL"
	EOF     Type = "EOF"
	NEWLINE Type = "NEWLINE"

	IDENT  Type = "IDENT"
	INT    Type = "INT"
	FLOAT  Type = "FLOAT"
	STRING Type = "STRING"

	ASSIGN   Type = "ASSIGN"
	PLUS_EQ  Type = "PLUS_EQ"
	MINUS_EQ Type = "MINUS_EQ"
	STAR_EQ  Type = "STAR_EQ"
	SLASH_EQ Type = "SLASH_EQ"
	INC      Type = "INC"
	DEC      Type = "DEC"

	PLUS    Type = "PLUS"
	MINUS   Type = "MINUS"
	STAR    Type = "STAR"
	SLASH   Type = "SLASH"
	PERCENT Type = "PERCENT"

	EQ  Type = "EQ"
	NEQ Type = "NEQ"
	LT  Type = "LT"
	LTE Type = "LTE"
	GT  Type = "GT"
	GTE Type = "GTE"

	AND Type = "AND"
	OR  Type = "OR"
	NOT Type = "NOT"

	LPAREN Type = "LPAREN"
	RPAREN Type = "RPAREN"
	LBRACK Type = "LBRACK"
	RBRACK Type = "RBRACK"
	LBRACE Type = "LBRACE"
	RBRACE Type = "RBRACE"
	COMMA  Type = "COMMA"
	TAG    Type = "TAG"
	QUEST  Type = "QUEST"
	ARROW  Type = "ARROW"
	METHOD Type = "METHOD"
	SELF   Type = "SELF"

	DEFINE Type = "DEFINE"
	VAR    Type = "VAR"
	CONST  Type = "CONST"

	TYPE_INT    Type = "TYPE_INT"
	TYPE_FLOAT  Type = "TYPE_FLOAT"
	TYPE_STRING Type = "TYPE_STRING"
	TYPE_BOOL   Type = "TYPE_BOOL"
	TYPE_ARRAY  Type = "TYPE_ARRAY"
	TYPE_DICT   Type = "TYPE_DICT"
	TYPE_ERROR  Type = "TYPE_ERROR"

	TRUE  Type = "TRUE"
	FALSE Type = "FALSE"
	NIL   Type = "NIL"

	IF      Type = "IF"
	SWITCH  Type = "SWITCH"
	CASE    Type = "CASE"
	DEFAULT Type = "DEFAULT"
	THEN    Type = "THEN"
	ELSE    Type = "ELSE"
	ELSEIF  Type = "ELSEIF"
	END     Type = "END"

	WHILE    Type = "WHILE"
	LOOP     Type = "LOOP"
	REPEAT   Type = "REPEAT"
	UNTIL    Type = "UNTIL"
	TIMES    Type = "TIMES"
	BREAK    Type = "BREAK"
	CONTINUE Type = "CONTINUE"
	FORIN    Type = "FORIN"
	INOF     Type = "INOF"
	FUNCTION Type = "FUNCTION"
	RETURN   Type = "RETURN"
	IMPORT   Type = "IMPORT"
	EXPORT   Type = "EXPORT"
	TRY      Type = "TRY"
	CATCH    Type = "CATCH"
	CHECK    Type = "CHECK"
	MEMBER   Type = "MEMBER"
	CLASS    Type = "CLASS"
	EMBED    Type = "EMBED"
	DOT      Type = "DOT"
	COLON    Type = "COLON"
)

type Token struct {
	Type    Type
	Literal string
	Line    int
	Column  int
}

var keywords = map[string]Type{
	"定义":   DEFINE,
	"变量":   VAR,
	"常量":   CONST,
	"整数":   TYPE_INT,
	"小数":   TYPE_FLOAT,
	"字符串":  TYPE_STRING,
	"布尔":   TYPE_BOOL,
	"数组":   TYPE_ARRAY,
	"字典":   TYPE_DICT,
	"错误":   TYPE_ERROR,
	"真":    TRUE,
	"假":    FALSE,
	"空":    NIL,
	"为":    ASSIGN,
	"加等于":  PLUS_EQ,
	"减等于":  MINUS_EQ,
	"乘等于":  STAR_EQ,
	"除等于":  SLASH_EQ,
	"自增":   INC,
	"自减":   DEC,
	"加":    PLUS,
	"减":    MINUS,
	"乘":    STAR,
	"除":    SLASH,
	"取余":   PERCENT,
	"等于":   EQ,
	"不等于":  NEQ,
	"小于":   LT,
	"小于等于": LTE,
	"大于":   GT,
	"大于等于": GTE,
	"且":    AND,
	"或":    OR,
	"非":    NOT,
	"判断":   SWITCH,
	"其他":   DEFAULT,
	"如果":   IF,
	"那么":   THEN,
	"否则":   ELSE,
	"否则如果": ELSEIF,
	"结束":   END,
	"当":    WHILE,
	"循环":   LOOP,
	"重复":   REPEAT,
	"直到":   UNTIL,
	"次":    TIMES,
	"跳出":   BREAK,
	"继续":   CONTINUE,
	"遍历":   FORIN,
	"中的":   INOF,
	"函数":   FUNCTION,
	"返回":   RETURN,
	"导入":   IMPORT,
	"导出":   EXPORT,
	"尝试":   TRY,
	"捕获":   CATCH,
	"检查":   CHECK,
	"的":    MEMBER,
	"方法":   METHOD,
	"自己":   SELF,
	"类":    CLASS,
	"模型":   CLASS,
	"嵌入":   EMBED,
}

func Lookup(ident string) (Type, bool) {
	t, ok := keywords[ident]
	return t, ok
}

func IsKeyword(ident string) bool {
	_, ok := keywords[ident]
	return ok
}
