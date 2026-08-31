package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/n8node/aiapp/internal/service"
)

type errorBody struct {
	Error errorPayload `json:"error"`
}

type errorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeError(w http.ResponseWriter, code int, publicCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(errorBody{Error: errorPayload{Code: publicCode, Message: message}})
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}

func mapAuthError(err error) (int, string, string) {
	var bitrixAPI *service.BitrixAPIError
	if errors.As(err, &bitrixAPI) && bitrixAPI.Public != "" {
		return http.StatusBadGateway, "bitrix_request", bitrixAPI.Public
	}
	switch {
	case errors.Is(err, service.ErrInvalidCredentials):
		return http.StatusUnauthorized, "invalid_credentials", "Неверный email или пароль"
	case errors.Is(err, service.ErrEmailTaken):
		return http.StatusConflict, "email_taken", "Не удалось создать учётную запись"
	case errors.Is(err, service.ErrUserBlocked):
		return http.StatusForbidden, "user_blocked", "Учётная запись заблокирована"
	case errors.Is(err, service.ErrInvalidInput):
		return http.StatusBadRequest, "invalid_input", "Проверьте введённые данные"
	case errors.Is(err, service.ErrInviteRequired):
		return http.StatusBadRequest, "invite_required", "Для регистрации нужен инвайт-ключ"
	case errors.Is(err, service.ErrInviteInvalid):
		return http.StatusBadRequest, "invite_invalid", "Инвайт-ключ недействителен"
	case errors.Is(err, service.ErrDomainNotAllowed):
		return http.StatusBadRequest, "domain_not_allowed", "Регистрация с этого почтового домена недоступна"
	case errors.Is(err, service.ErrEmailReserved):
		return http.StatusBadRequest, "email_reserved", "Этот адрес зарезервирован"
	case errors.Is(err, service.ErrTotpRequired):
		return http.StatusUnauthorized, "totp_required", "Введите код двухфакторной аутентификации"
	case errors.Is(err, service.ErrTotpSetupRequired):
		return http.StatusForbidden, "totp_setup_required", "Настройте двухфакторную аутентификацию"
	case errors.Is(err, service.ErrInvalidTotp):
		return http.StatusUnauthorized, "invalid_totp", "Неверный код двухфакторной аутентификации"
	case errors.Is(err, service.ErrTotpAlreadyEnabled):
		return http.StatusConflict, "totp_enabled", "Двухфакторная аутентификация уже включена"
	case errors.Is(err, service.ErrForbidden):
		return http.StatusForbidden, "forbidden", "Недостаточно прав"
	case errors.Is(err, service.ErrBitrixNotConfigured):
		return http.StatusBadRequest, "bitrix_not_configured", "Сначала сохраните вебхук Битрикс24"
	case errors.Is(err, service.ErrBitrixInvalidURL):
		return http.StatusBadRequest, "bitrix_invalid_url", "Укажите HTTPS URL входящего вебхука вида https://портал/rest/1/ключ/"
	case errors.Is(err, service.ErrBitrixBlockedHost):
		return http.StatusBadRequest, "bitrix_blocked_host", "Этот адрес вебхука нельзя использовать"
	case errors.Is(err, service.ErrBitrixRequest):
		return http.StatusBadGateway, "bitrix_request", "Битрикс24 не ответил или отклонил запрос. Проверьте права вебхука."
	case errors.Is(err, service.ErrBitrixBusy):
		return http.StatusConflict, "bitrix_busy", "Синхронизация уже выполняется"
	case errors.Is(err, service.ErrWorkspaceNotFound):
		return http.StatusNotFound, "workspace_not_found", "Рабочее пространство не найдено"
	case errors.Is(err, service.ErrDepartmentUnknown):
		return http.StatusBadRequest, "department_unknown", "Сначала синхронизируйте отделы из Битрикс24"
	case errors.Is(err, service.ErrDepartmentMapped):
		return http.StatusConflict, "department_mapped", "Этот отдел уже привязан к пространству"
	case errors.Is(err, service.ErrInvalidStorageSettings):
		return http.StatusBadRequest, "invalid_storage", "Проверьте настройки S3: endpoint, бакет и ключи доступа"
	case errors.Is(err, service.ErrStorageNotConfigured):
		return http.StatusBadRequest, "storage_not_configured", "Администратор ещё не подключил хранилище"
	case errors.Is(err, service.ErrDiskNoWorkspace):
		return http.StatusBadRequest, "no_workspace", "Сначала выберите пространство отдела"
	case errors.Is(err, service.ErrDiskFileNotFound):
		return http.StatusNotFound, "file_not_found", "Файл не найден"
	case errors.Is(err, service.ErrDiskFolderNotFound):
		return http.StatusNotFound, "folder_not_found", "Папка не найдена"
	case errors.Is(err, service.ErrDiskFileTooLarge):
		return http.StatusRequestEntityTooLarge, "file_too_large", "Файл слишком большой"
	case errors.Is(err, service.ErrDiskEmptyFile):
		return http.StatusBadRequest, "empty_file", "Пустой файл загружать нельзя"
	case errors.Is(err, service.ErrDiskTypeRejected):
		return http.StatusBadRequest, "file_type", "Этот тип файла не принимается"
	case errors.Is(err, service.ErrDiskInvalidMove):
		return http.StatusBadRequest, "invalid_move", "Нельзя переместить папку саму в себя"
	case errors.Is(err, service.ErrDiskNameTaken):
		return http.StatusConflict, "name_taken", "Папка с таким именем уже есть"
	case errors.Is(err, service.ErrDiskUploadSession):
		return http.StatusBadRequest, "upload_session", "Загрузка не завершена. Повторите попытку."
	case errors.Is(err, service.ErrDiskStorageDelete):
		return http.StatusBadGateway, "storage_delete", "Не удалось удалить файл из хранилища. Повторите попытку."
	case errors.Is(err, service.ErrDiskStorageWrite):
		return http.StatusBadGateway, "storage_write", "Не удалось записать файл в хранилище. Повторите попытку."
	case errors.Is(err, service.ErrDiskNotArchive):
		return http.StatusBadRequest, "not_archive", "Этот файл нельзя распаковать"
	case errors.Is(err, service.ErrDiskArchiveLimit):
		return http.StatusBadRequest, "archive_limit", "Архив слишком большой или в нём слишком много файлов"
	case errors.Is(err, service.ErrDiskArchiveEncrypt):
		return http.StatusBadRequest, "archive_encrypted", "Архив защищён паролем"
	default:
		return http.StatusInternalServerError, "internal_error", "Не удалось выполнить запрос"
	}
}

func writeAuthError(w http.ResponseWriter, err error) {
	code, public, msg := mapAuthError(err)
	writeError(w, code, public, msg)
}
