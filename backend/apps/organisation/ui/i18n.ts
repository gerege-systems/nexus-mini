// organisation модулийн толь — түлхүүр нь монгол текст (цөмийн i18n-тэй ижил
// дүрэм), хэл бүр нэг объект. Portal build үед scripts/sync-modules.mjs
// цөмийн толинд нэгтгэнэ; цөмийн файлд гар хүрэхгүй.
const i18n: Record<string, Record<string, string>> = {
  en: {
    "Ажилтан": "People",
    "Ажилтнууд": "People",
    "Албан тушаал": "Job title",
    "Алдаа гарлаа": "Something went wrong",
    "Байгууллагын бүтцийн мод": "Organization structure tree",
    "Болих": "Cancel",
    "Гишүүн байхгүй": "No members",
    "Гишүүн бүрийн хэлтэс, албан тушаал": "Department and job title of every member",
    "Дээд нэгж": "Parent unit",
    "Идэвхтэй": "Active",
    "Имэйл": "Email",
    "Код": "Code",
    "Менежер": "Manager",
    "Нэгж": "Unit",
    "Нэгж байхгүй": "No units yet",
    "Нэгж засах": "Edit unit",
    "Нэгж нэмэх": "Add unit",
    "Нэр": "Name",
    "Устгагдлаа": "Deleted",
    "Хадгалагдлаа": "Saved",
    "Хадгалах": "Save",
    "Хэлтэс": "Department",
    "Хэлтэс, нэгж": "Departments",
    "Эхний хэлтэс/нэгжээ үүсгээрэй": "Create your first department or unit",
    "засах": "edit",
    "идэвхгүй": "inactive",
    "устгах": "delete",
    "хэлтсийг устгах уу? Харьяа нэгжүүд дээд түвшингүй болно.": "— delete this unit? Child units become top-level.",
    "— байхгүй (дээд түвшин) —": "— none (top level) —",
  },
};

export default i18n;
