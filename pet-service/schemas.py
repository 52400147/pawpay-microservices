from pydantic import BaseModel, Field


class PetProfileBase(BaseModel):
    name: str = Field(..., min_length=1, max_length=100, description="Tên thú cưng")
    user_id: int = Field(..., description="ID chủ nuôi (bên User Service)")


class PetProfileCreate(PetProfileBase):
    pass


class PetProfileUpdate(BaseModel):
    name: str
    user_id: int


class PetProfileOut(PetProfileBase):
    id: int

    class Config:
        from_attributes = True
