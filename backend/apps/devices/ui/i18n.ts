// devices модулийн толь — түлхүүр нь монгол текст (цөмийн i18n-тэй ижил
// дүрэм), хэл бүр нэг объект. Portal build үед scripts/sync-modules.mjs
// цөмийн толинд нэгтгэнэ; цөмийн файлд гар хүрэхгүй.
const i18n: Record<string, Record<string, string>> = {
  en: {
    "Илэрц алга": "No matches",
    "Уншиж байна…": "Loading…",
    "Алдаа гарлаа": "Something went wrong",
    "Алдагдсан": "Lost",
    "Ашиглагдаж байгаа": "In use",
    "Байгууллагын төхөөрөмжийн бүртгэл": "Your organization's device registry",
    "Болих": "Cancel",
    "Бүртгэгдлээ": "Registered",
    "Бүртгэл хоосон": "No records yet",
    "Бүртгэсэн": "Registered by",
    "Бүртгэх": "Register",
    "Засварт": "In repair",
    "Нэр": "Name",
    "Сериал": "Serial",
    "Статус": "Status",
    "Тэмдэглэл": "Note",
    "Төрөл": "Type",
    "Төхөөрөмж бүртгэх": "Register device",
    "Төхөөрөмж засах": "Edit device",
    "Төхөөрөмжүүд": "Devices",
    "Устгагдлаа": "Deleted",
    "Хадгалагдлаа": "Saved",
    "Хадгалах": "Save",
    "Хайх…": "Search…",
    "Хассан": "Retired",
    "Эхний төхөөрөмжөө бүртгээрэй": "Register your first device",
    "засах": "edit",
    "төхөөрөмжийг устгах уу?": "— delete this device?",
    "устгах": "delete",
  },
};

export default i18n;
