package dto

type InsertStaff struct {
	FirstName string `json:"firstname"`
	Lastname  string `json:"lastname"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	Role      string `json:"role"`
}

type StaffModel struct {
	StaffId   int    `json:"staff_id" db:"staff_id"`
	FirstName string `json:"firstname" db:"firstname"`
	Lastname  string `json:"lastname" db:"lastname"`
	Email     string `json:"email" db:"email"`
	Password  string `json:"password" db:"password"`
	Role      string `json:"role" db:"role"`
}
