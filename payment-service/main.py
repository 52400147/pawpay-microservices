from typing import List, Optional

from fastapi import Depends, FastAPI, Form, HTTPException, Request
from fastapi.responses import HTMLResponse, RedirectResponse
from fastapi.templating import Jinja2Templates
from sqlalchemy.orm import Session

import models
import schemas
from database import Base, engine, get_db

Base.metadata.create_all(bind=engine)

app = FastAPI(title="Pet Profile Service", version="1.0.0")
templates = Jinja2Templates(directory="templates")


# ============================================================
#  REST API — dùng cho Gateway và các service khác gọi vào
# ============================================================

@app.post("/api/pets", response_model=schemas.PetProfileOut, status_code=201)
def create_pet(pet: schemas.PetProfileCreate, db: Session = Depends(get_db)):
    """Tạo mới 1 hồ sơ thú cưng."""
    db_pet = models.PetProfile(**pet.model_dump())
    db.add(db_pet)
    db.commit()
    db.refresh(db_pet)
    return db_pet


@app.get("/api/pets", response_model=List[schemas.PetProfileOut])
def list_pets(user_id: Optional[int] = None, db: Session = Depends(get_db)):
    """Lấy danh sách hồ sơ thú cưng. Có thể lọc theo user_id (tra cứu theo chủ nuôi)."""
    query = db.query(models.PetProfile)
    if user_id is not None:
        query = query.filter(models.PetProfile.user_id == user_id)
    return query.all()


@app.get("/api/pets/{pet_id}", response_model=schemas.PetProfileOut)
def get_pet(pet_id: int, db: Session = Depends(get_db)):
    """Tra cứu chi tiết 1 hồ sơ thú cưng theo id."""
    pet = db.query(models.PetProfile).filter(models.PetProfile.id == pet_id).first()
    if not pet:
        raise HTTPException(status_code=404, detail="Không tìm thấy hồ sơ thú cưng")
    return pet


@app.put("/api/pets/{pet_id}", response_model=schemas.PetProfileOut)
def update_pet(pet_id: int, pet_update: schemas.PetProfileUpdate, db: Session = Depends(get_db)):
    """Cập nhật thông tin hồ sơ thú cưng."""
    pet = db.query(models.PetProfile).filter(models.PetProfile.id == pet_id).first()
    if not pet:
        raise HTTPException(status_code=404, detail="Không tìm thấy hồ sơ thú cưng")
    for key, value in pet_update.model_dump(exclude_unset=True).items():
        setattr(pet, key, value)
    db.commit()
    db.refresh(pet)
    return pet


@app.delete("/api/pets/{pet_id}", status_code=204)
def delete_pet(pet_id: int, db: Session = Depends(get_db)):
    """Xóa 1 hồ sơ thú cưng."""
    pet = db.query(models.PetProfile).filter(models.PetProfile.id == pet_id).first()
    if not pet:
        raise HTTPException(status_code=404, detail="Không tìm thấy hồ sơ thú cưng")
    db.delete(pet)
    db.commit()
    return None


# ============================================================
#  Trang Admin — quản lý hồ sơ thú cưng bằng giao diện web
# ============================================================

@app.get("/admin/pets", response_class=HTMLResponse)
def admin_list(request: Request, db: Session = Depends(get_db)):
    pets = db.query(models.PetProfile).order_by(models.PetProfile.id.desc()).all()
    return templates.TemplateResponse("admin.html", {"request": request, "pets": pets})


@app.post("/admin/pets")
def admin_create(
    name: str = Form(...),
    species: str = Form(""),
    user_id: int = Form(...),
    db: Session = Depends(get_db),
):
    db_pet = models.PetProfile(name=name, species=species or None, user_id=user_id)
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
def admin_edit(
    pet_id: int,
    name: str = Form(...),
    species: str = Form(""),
    db: Session = Depends(get_db),
):
    pet = db.query(models.PetProfile).filter(models.PetProfile.id == pet_id).first()
    if not pet:
        raise HTTPException(status_code=404, detail="Không tìm thấy hồ sơ thú cưng")
    pet.name = name
    pet.species = species or None
    db.commit()
    return RedirectResponse(url="/admin/pets", status_code=303)


@app.post("/admin/pets/{pet_id}/delete")
def admin_delete(pet_id: int, db: Session = Depends(get_db)):
    pet = db.query(models.PetProfile).filter(models.PetProfile.id == pet_id).first()
    if pet:
        db.delete(pet)
        db.commit()
    return RedirectResponse(url="/admin/pets", status_code=303)
