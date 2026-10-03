from typing import List

from fastapi import Depends, FastAPI, Form, HTTPException, Request
from fastapi.exceptions import RequestValidationError
from fastapi.responses import HTMLResponse, JSONResponse, RedirectResponse
from fastapi.templating import Jinja2Templates
from sqlalchemy.orm import Session

import models
import schemas
from auth import get_current_user, require_admin
from database import Base, engine, get_db

Base.metadata.create_all(bind=engine)

app = FastAPI(title="Pet Profile Service", version="1.0.0")
templates = Jinja2Templates(directory="templates")


# ============================================================
#  Chuẩn hóa lỗi: LUÔN trả {"message": "..."} theo quy ước chung
# ============================================================

@app.exception_handler(HTTPException)
async def http_exception_handler(request: Request, exc: HTTPException):
    return JSONResponse(status_code=exc.status_code, content={"message": exc.detail})


@app.exception_handler(RequestValidationError)
async def validation_exception_handler(request: Request, exc: RequestValidationError):
    return JSONResponse(status_code=400, content={"message": "Dữ liệu gửi lên không hợp lệ"})


# ============================================================
#  REST API — port 3002, dùng cho Gateway và các service khác gọi
#  Theo đúng hợp đồng API: /pets/{code}, /pets, /pets/{id}
# ============================================================

@app.get("/pets/{code}", response_model=schemas.PetProfileOut)
def get_pet_by_code(code: str, db: Session = Depends(get_db), user: dict = Depends(get_current_user)):
    """Tra cứu 1 hồ sơ theo mã (hiện dùng id làm code). Cần token hợp lệ, không cần admin."""
    try:
        pet_id = int(code)
    except ValueError:
        raise HTTPException(status_code=404, detail="Không tìm thấy hồ sơ thú cưng")

    pet = db.query(models.PetProfile).filter(models.PetProfile.id == pet_id).first()
    if not pet:
        raise HTTPException(status_code=404, detail="Không tìm thấy hồ sơ thú cưng")
    return pet


@app.get("/pets", response_model=List[schemas.PetProfileOut])
def list_pets(db: Session = Depends(get_db), user: dict = Depends(require_admin)):
    """Danh sách toàn bộ hồ sơ thú cưng — chỉ admin."""
    return db.query(models.PetProfile).all()


@app.post("/pets", response_model=schemas.PetProfileOut, status_code=201)
def create_pet(pet: schemas.PetProfileCreate, db: Session = Depends(get_db), user: dict = Depends(require_admin)):
    """Tạo mới hồ sơ thú cưng — chỉ admin."""
    db_pet = models.PetProfile(**pet.model_dump())
    db.add(db_pet)
    db.commit()
    db.refresh(db_pet)
    return db_pet


@app.put("/pets/{pet_id}", response_model=schemas.PetProfileOut)
def update_pet(pet_id: int, pet_update: schemas.PetProfileUpdate, db: Session = Depends(get_db), user: dict = Depends(require_admin)):
    """Cập nhật hồ sơ thú cưng — chỉ admin."""
    pet = db.query(models.PetProfile).filter(models.PetProfile.id == pet_id).first()
    if not pet:
        raise HTTPException(status_code=404, detail="Không tìm thấy hồ sơ thú cưng")
    pet.name = pet_update.name
    pet.user_id = pet_update.user_id
    db.commit()
    db.refresh(pet)
    return pet


@app.delete("/pets/{pet_id}", status_code=200)
def delete_pet(pet_id: int, db: Session = Depends(get_db), user: dict = Depends(require_admin)):
    """Xóa hồ sơ thú cưng — chỉ admin. Trả 200 theo đúng hợp đồng (không phải 204)."""
    pet = db.query(models.PetProfile).filter(models.PetProfile.id == pet_id).first()
    if not pet:
        raise HTTPException(status_code=404, detail="Không tìm thấy hồ sơ thú cưng")
    db.delete(pet)
    db.commit()
    return {"message": "Đã xóa hồ sơ thú cưng"}


# ============================================================
#  Trang Admin web — công cụ riêng để Nhung tự quản lý/test dữ liệu
#  (không nằm trong hợp đồng API, không cần JWT, chỉ chạy nội bộ)
# ============================================================

@app.get("/admin/pets", response_class=HTMLResponse)
def admin_list(request: Request, db: Session = Depends(get_db)):
    pets = db.query(models.PetProfile).order_by(models.PetProfile.id.desc()).all()
    return templates.TemplateResponse("admin.html", {"request": request, "pets": pets})


@app.post("/admin/pets")
def admin_create(name: str = Form(...), user_id: int = Form(...), db: Session = Depends(get_db)):
    db_pet = models.PetProfile(name=name, user_id=user_id)
    db.add(db_pet)
    db.commit()
    return RedirectResponse(url="/admin/pets", status_code=303)


@app.get("/admin/pets/{pet_id}/edit", response_class=HTMLResponse)
def admin_edit_form(pet_id: int, request: Request, db: Session = Depends(get_db)):
    pet = db.query(models.PetProfile).filter(models.PetProfile.id == pet_id).first()
    if not pet:
        raise HTTPException(status_code=404, detail="Không tìm thấy hồ sơ thú cưng")
    return templates.TemplateResponse("edit.html", {"request": request, "pet": pet})


@app.post("/admin/pets/{pet_id}/edit")
def admin_edit(pet_id: int, name: str = Form(...), user_id: int = Form(...), db: Session = Depends(get_db)):
    pet = db.query(models.PetProfile).filter(models.PetProfile.id == pet_id).first()
    if not pet:
        raise HTTPException(status_code=404, detail="Không tìm thấy hồ sơ thú cưng")
    pet.name = name
    pet.user_id = user_id
    db.commit()
    return RedirectResponse(url="/admin/pets", status_code=303)


@app.post("/admin/pets/{pet_id}/delete")
def admin_delete(pet_id: int, db: Session = Depends(get_db)):
    pet = db.query(models.PetProfile).filter(models.PetProfile.id == pet_id).first()
    if pet:
        db.delete(pet)
        db.commit()
    return RedirectResponse(url="/admin/pets", status_code=303)
