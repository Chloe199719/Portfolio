interface Sanitybody {
  _createdAt: string;
  _id: string;
  _rev: string;
  _updatedAt: string;
}
interface Image {
  _type: "image";
  asset: {
    _ref: string;
    _type: "reference";
  };
}
export interface PageInfo extends Sanitybody {
  _type: "pageInfo";
  address: string;
  backgroundInformation: string;
  email: string;
  heroImage: Image;
  name: string;
  phoneNumber: string;
  profilePic: Image;
}
export interface Social extends Sanitybody {
  _type: "social";
  title: string;
  url: string;
}

export interface Skill extends Sanitybody {
  _type: "skill";
  image: Image;
  progress: number;
  title: string;
}
export interface Technology extends Sanitybody {
  _type: "skill";
  image: Image;
  progress: number;
  title: string;
}

export interface Project extends Sanitybody {
  title: string;
  _type: "project";
  image: Image;
  linkToBuild: String;
  summary: string;
  technologies: Technology[];
}

export interface Experience extends Sanitybody {
  _type: "experience";
  company: string;
  companyImage: Image;
  dataStarted: date;
  dataEnded: date;
  isCurrentlyWorkingHere: boolean;
  jobTitle: String;
  points: string[];
  technologies: Technology[];
}
